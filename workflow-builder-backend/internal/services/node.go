package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wf-builder/workflow-builder-backend/internal/models"
)

var (
	ErrNotFound      = errors.New("not found")
	ErrConflict      = errors.New("conflict")
	ErrInvalidInput  = errors.New("invalid input")
)

type NodeService struct {
	pool      *pgxpool.Pool
	workflows *WorkflowService
}

func NewNodeService(pool *pgxpool.Pool, workflows *WorkflowService) *NodeService {
	return &NodeService{pool: pool, workflows: workflows}
}

type AddNodeInput struct {
	Label     string  `json:"label"`
	Type      string  `json:"type"`
	PositionX float64 `json:"positionX"`
	PositionY float64 `json:"positionY"`
}

func (s *NodeService) Add(ctx context.Context, workflowID string, in AddNodeInput) (map[string]any, error) {
	if in.Label == "" || in.Type == "" {
		return nil, fmt.Errorf("%w: missing required parameters", ErrInvalidInput)
	}

	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	nodeID := uuid.NewString()
	now := time.Now().UTC()
	nodeData := map[string]any{}

	if in.Type == string(models.NodeTypeWebhook) {
		webhookPath := uuid.NewString()
		nodeData = map[string]any{
			"parameters": map[string]any{
				"method":      "GET",
				"path":        webhookPath,
				"respondType": "IMMEDIATELY",
			},
			"settings": map[string]any{
				"allowMultipleHttps": false,
				"notes":              "",
			},
		}
	}

	dataBytes, err := json.Marshal(nodeData)
	if err != nil {
		return nil, err
	}
	// Store as JSON string to match existing Prisma string-stored JSON behavior
	// when some rows have stringified JSON; also accept raw JSONB.
	dataStr := string(dataBytes)

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		INSERT INTO "workflowNodes"
			(id, "workflowId", type, "positionX", "positionY", label, data, "createdAt", "updatedAt")
		VALUES ($1, $2, $3, $4, $5, $6, $7::jsonb, $8, $9)
	`, nodeID, workflowID, in.Type, in.PositionX, in.PositionY, in.Label, dataStr, now, now)
	if err != nil {
		return nil, err
	}

	if in.Type == string(models.NodeTypeWebhook) {
		// Match Next.js behavior: path stored as nodeId on webhook row.
		var existing string
		err = tx.QueryRow(ctx, `
			SELECT id FROM webhooks WHERE "workflowId" = $1 AND path = $2 LIMIT 1
		`, workflowID, nodeID).Scan(&existing)
		if err == nil {
			return nil, fmt.Errorf("%w: webhook path already exists", ErrConflict)
		}
		if err != nil && !errors.Is(err, pgx.ErrNoRows) {
			return nil, err
		}

		_, err = tx.Exec(ctx, `
			INSERT INTO webhooks (id, path, "workflowId", "createdAt", "updatedAt", method, "nodeId")
			VALUES ($1, $2, $3, $4, $5, 'GET', $6)
		`, uuid.NewString(), nodeID, workflowID, now, now, nodeID)
		if err != nil {
			return nil, err
		}
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return map[string]any{
		"id":   nodeID,
		"type": in.Type,
		"position": map[string]any{
			"positionX": in.PositionX,
			"positionY": in.PositionY,
		},
		"data": map[string]any{
			"label":       in.Label,
			"description": "",
		},
	}, nil
}

func (s *NodeService) Delete(ctx context.Context, workflowID, nodeID string) (map[string]any, error) {
	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	var (
		nType models.NodeType
		data  []byte
	)
	err = s.pool.QueryRow(ctx, `
		SELECT type, data FROM "workflowNodes" WHERE id = $1
	`, nodeID).Scan(&nType, &data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return nil, err
	}
	defer tx.Rollback(ctx)

	_, err = tx.Exec(ctx, `
		DELETE FROM "workflowEdges" WHERE source = $1 OR target = $1
	`, nodeID)
	if err != nil {
		return nil, err
	}

	if nType == models.NodeTypeWebhook {
		_, _ = tx.Exec(ctx, `DELETE FROM webhooks WHERE "nodeId" = $1`, nodeID)
	}

	var deletedID string
	err = tx.QueryRow(ctx, `
		DELETE FROM "workflowNodes" WHERE id = $1 RETURNING id
	`, nodeID).Scan(&deletedID)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	if err := tx.Commit(ctx); err != nil {
		return nil, err
	}

	return map[string]any{
		"id": deletedID,
	}, nil
}

type GraphNodeInput struct {
	ID       string `json:"id"`
	Position struct {
		X float64 `json:"x"`
		Y float64 `json:"y"`
	} `json:"position"`
	Data struct {
		Label       string `json:"label"`
		Icon        string `json:"icon"`
		Color       string `json:"color"`
		Description string `json:"description"`
	} `json:"data"`
}

type GraphEdgeInput struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

func (s *NodeService) UpdateNodesAndEdges(ctx context.Context, workflowID string, nodes []GraphNodeInput, edges []GraphEdgeInput) error {
	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return err
	}
	if w == nil {
		return ErrNotFound
	}

	tx, err := s.pool.Begin(ctx)
	if err != nil {
		return err
	}
	defer tx.Rollback(ctx)

	if _, err := tx.Exec(ctx, `DELETE FROM "workflowEdges" WHERE "workflowId" = $1`, workflowID); err != nil {
		return err
	}

	now := time.Now().UTC()
	for _, edge := range edges {
		edgeID := edge.ID
		if edgeID == "" {
			edgeID = uuid.NewString()
		}
		_, err := tx.Exec(ctx, `
			INSERT INTO "workflowEdges" (id, "workflowId", source, target, "createdAt", "updatedAt")
			VALUES ($1, $2, $3, $4, $5, $6)
		`, edgeID, workflowID, edge.Source, edge.Target, now, now)
		if err != nil {
			return err
		}
	}

	for _, node := range nodes {
		_, err := tx.Exec(ctx, `
			UPDATE "workflowNodes"
			SET "positionX" = $2, "positionY" = $3, color = $4, label = $5, icon = $6, description = $7, "updatedAt" = $8
			WHERE id = $1
		`, node.ID, node.Position.X, node.Position.Y, node.Data.Color, node.Data.Label, node.Data.Icon, node.Data.Description, now)
		if err != nil {
			return err
		}
	}

	return tx.Commit(ctx)
}

func (s *NodeService) UpdateNodeData(ctx context.Context, workflowID, nodeID string, data any) (map[string]any, error) {
	var nType models.NodeType
	err := s.pool.QueryRow(ctx, `
		SELECT type FROM "workflowNodes" WHERE id = $1 AND "workflowId" = $2
	`, nodeID, workflowID).Scan(&nType)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	dataMap, _ := data.(map[string]any)
	if nType == models.NodeTypeWebhook && dataMap != nil {
		params, _ := dataMap["parameters"].(map[string]any)
		path, _ := params["path"].(string)
		method, _ := params["method"].(string)
		if method == "" {
			method = "GET"
		}

		if path != "" {
			rows, err := s.pool.Query(ctx, `SELECT id, "workflowId", method, path FROM webhooks WHERE path = $1`, path)
			if err != nil {
				return nil, err
			}
			for rows.Next() {
				var wid, wfID, m, p string
				if err := rows.Scan(&wid, &wfID, &m, &p); err != nil {
					rows.Close()
					return nil, err
				}
				if wfID != workflowID && m == method && p == path {
					rows.Close()
					return nil, fmt.Errorf("%w: webhook path and method combination already exists in another workflow", ErrConflict)
				}
			}
			rows.Close()
			if err := rows.Err(); err != nil {
				return nil, err
			}
		}

		var existingNodeID string
		err = s.pool.QueryRow(ctx, `SELECT "nodeId" FROM webhooks WHERE "nodeId" = $1`, nodeID).Scan(&existingNodeID)
		now := time.Now().UTC()
		if errors.Is(err, pgx.ErrNoRows) {
			_, err = s.pool.Exec(ctx, `
				INSERT INTO webhooks (id, path, "workflowId", "createdAt", "updatedAt", method, "nodeId")
				VALUES ($1, $2, $3, $4, $5, $6, $7)
			`, uuid.NewString(), path, workflowID, now, now, method, nodeID)
			if err != nil {
				return nil, err
			}
		} else if err != nil {
			return nil, err
		} else {
			_, err = s.pool.Exec(ctx, `
				UPDATE webhooks SET path = $2, method = $3, "workflowId" = $4, "updatedAt" = $5
				WHERE "nodeId" = $1
			`, nodeID, path, method, workflowID, now)
			if err != nil {
				return nil, err
			}
		}
	}

	dataBytes, err := json.Marshal(data)
	if err != nil {
		return nil, err
	}
	now := time.Now().UTC()
	var updatedID string
	err = s.pool.QueryRow(ctx, `
		UPDATE "workflowNodes" SET data = $2::jsonb, "updatedAt" = $3
		WHERE id = $1
		RETURNING id
	`, nodeID, string(dataBytes), now).Scan(&updatedID)
	if err != nil {
		return nil, err
	}

	return map[string]any{
		"id":   updatedID,
		"data": json.RawMessage(dataBytes),
	}, nil
}

func (s *NodeService) GetNodeData(ctx context.Context, workflowID, nodeID string) (any, error) {
	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	var data []byte
	err = s.pool.QueryRow(ctx, `
		SELECT data FROM "workflowNodes" WHERE id = $1 AND "workflowId" = $2
	`, nodeID, workflowID).Scan(&data)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}
	if len(data) == 0 {
		return map[string]any{}, nil
	}

	// data may be raw JSON or a JSON-encoded string
	var asAny any
	if err := json.Unmarshal(data, &asAny); err != nil {
		return map[string]any{}, nil
	}
	if s, ok := asAny.(string); ok {
		var nested any
		if err := json.Unmarshal([]byte(s), &nested); err == nil {
			return nested, nil
		}
		return map[string]any{}, nil
	}
	return asAny, nil
}
