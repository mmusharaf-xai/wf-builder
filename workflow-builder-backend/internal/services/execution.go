package services

import (
	"context"
	"encoding/json"
	"errors"
	"strconv"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wf-builder/workflow-builder-backend/internal/models"
)

type ExecutionService struct {
	pool      *pgxpool.Pool
	workflows *WorkflowService
}

func NewExecutionService(pool *pgxpool.Pool, workflows *WorkflowService) *ExecutionService {
	return &ExecutionService{pool: pool, workflows: workflows}
}

type ListExecutionsResult struct {
	CurrentPage  int                       `json:"current_page"`
	TotalPages   int                       `json:"total_pages"`
	TotalRecords int                       `json:"total_records"`
	Result       []models.ExecutionHistory `json:"result"`
}

type ListExecutionsFilter struct {
	// From is inclusive lower bound on createdAt (UTC).
	From *time.Time
	// To is inclusive upper bound on createdAt (UTC).
	To *time.Time
}

func (s *ExecutionService) List(ctx context.Context, workflowID string, page, pageSize int, filter ListExecutionsFilter) (*ListExecutionsResult, error) {
	if page < 1 {
		page = 1
	}
	if pageSize < 1 {
		pageSize = 10
	}
	if pageSize > 100 {
		pageSize = 100
	}

	w, err := s.workflows.GetByID(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, ErrNotFound
	}

	where := `"workflowId" = $1`
	args := []any{workflowID}
	argN := 2
	if filter.From != nil {
		where += ` AND "createdAt" >= $` + strconv.Itoa(argN)
		args = append(args, filter.From.UTC())
		argN++
	}
	if filter.To != nil {
		where += ` AND "createdAt" <= $` + strconv.Itoa(argN)
		args = append(args, filter.To.UTC())
		argN++
	}

	var total int
	countQ := `SELECT COUNT(*) FROM "executionsHistory" WHERE ` + where
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	totalPages := 0
	if pageSize > 0 {
		totalPages = (total + pageSize - 1) / pageSize
	}
	offset := (page - 1) * pageSize

	listQ := `
		SELECT id, "workflowId", status, "createdAt", "updatedAt", "completedAt"
		FROM "executionsHistory"
		WHERE ` + where + `
		ORDER BY "createdAt" DESC
		LIMIT $` + strconv.Itoa(argN) + ` OFFSET $` + strconv.Itoa(argN+1)
	args = append(args, pageSize, offset)

	rows, err := s.pool.Query(ctx, listQ, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	items := make([]models.ExecutionHistory, 0)
	for rows.Next() {
		var h models.ExecutionHistory
		if err := rows.Scan(&h.ID, &h.WorkflowID, &h.Status, &h.CreatedAt, &h.UpdatedAt, &h.CompletedAt); err != nil {
			return nil, err
		}
		items = append(items, h)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	return &ListExecutionsResult{
		CurrentPage:  page,
		TotalPages:   totalPages,
		TotalRecords: total,
		Result:       items,
	}, nil
}



func (s *ExecutionService) GetGraph(ctx context.Context, workflowID, executionID string) (*models.Workflow, []models.GraphNode, []models.ExecutionEdge, error) {
	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return nil, nil, nil, err
	}
	if w == nil {
		return nil, nil, nil, ErrNotFound
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, type, "positionX", "positionY", label, icon, color, description, "workflowNodeId", "executionId", "createdAt", "updatedAt"
		FROM "executionNodes" WHERE "executionId" = $1
	`, executionID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	hashed := map[string]models.GraphNode{}
	nodeIDs := make([]string, 0)
	for rows.Next() {
		var (
			id, label, workflowNodeID, execID string
			nType                             models.NodeType
			px, py                            float64
			icon, color, desc                 *string
			createdAt, updatedAt              time.Time
		)
		if err := rows.Scan(&id, &nType, &px, &py, &label, &icon, &color, &desc, &workflowNodeID, &execID, &createdAt, &updatedAt); err != nil {
			return nil, nil, nil, err
		}
		data := map[string]any{"label": label}
		if icon != nil {
			data["icon"] = *icon
		}
		if color != nil {
			data["color"] = *color
		}
		if desc != nil {
			data["description"] = *desc
		}
		hashed[id] = models.GraphNode{
			ID:             id,
			Type:           nType,
			WorkflowNodeID: workflowNodeID,
			Position:       models.Position{X: px, Y: py},
			Data:           data,
		}
		nodeIDs = append(nodeIDs, id)
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	// Aggregate status counts per execution node
	if len(nodeIDs) > 0 {
		statusRows, err := s.pool.Query(ctx, `
			SELECT "nodeId", status FROM executions
			WHERE "executionId" = $1 AND "nodeId" = ANY($2)
		`, executionID, nodeIDs)
		if err != nil {
			return nil, nil, nil, err
		}
		counts := map[string]map[string]int{}
		for statusRows.Next() {
			var nodeID string
			var status models.ExecutionStatus
			if err := statusRows.Scan(&nodeID, &status); err != nil {
				statusRows.Close()
				return nil, nil, nil, err
			}
			if counts[nodeID] == nil {
				counts[nodeID] = map[string]int{}
			}
			counts[nodeID][string(status)]++
		}
		statusRows.Close()
		if err := statusRows.Err(); err != nil {
			return nil, nil, nil, err
		}
		for id, node := range hashed {
			if c, ok := counts[id]; ok {
				node.Data["statusCounts"] = c
				hashed[id] = node
			}
		}
	}

	nodes := make([]models.GraphNode, 0, len(hashed))
	for _, n := range hashed {
		nodes = append(nodes, n)
	}

	edgeRows, err := s.pool.Query(ctx, `
		SELECT id, source, target, "executionId", "createdAt", "updatedAt"
		FROM "executionEdges" WHERE "executionId" = $1
	`, executionID)
	if err != nil {
		return nil, nil, nil, err
	}
	defer edgeRows.Close()

	edges := make([]models.ExecutionEdge, 0)
	for edgeRows.Next() {
		var e models.ExecutionEdge
		if err := edgeRows.Scan(&e.ID, &e.Source, &e.Target, &e.ExecutionID, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, nil, nil, err
		}
		edges = append(edges, e)
	}
	if err := edgeRows.Err(); err != nil {
		return nil, nil, nil, err
	}

	return w, nodes, edges, nil
}

func (s *ExecutionService) GetNodeData(ctx context.Context, workflowID, executionID, workflowNodeID string) (map[string]any, error) {
	w, err := s.workflows.GetOwned(ctx, workflowID)
	if err != nil {
		return nil, err
	}
	if w == nil || workflowNodeID == "" {
		return nil, ErrNotFound
	}

	var (
		execNodeID string
		rawData    []byte
	)
	err = s.pool.QueryRow(ctx, `
		SELECT id, data FROM "executionNodes"
		WHERE "executionId" = $1 AND "workflowNodeId" = $2
	`, executionID, workflowNodeID).Scan(&execNodeID, &rawData)
	if errors.Is(err, pgx.ErrNoRows) {
		return nil, ErrNotFound
	}
	if err != nil {
		return nil, err
	}

	nodeData := map[string]any{}
	if len(rawData) > 0 {
		var asAny any
		if err := json.Unmarshal(rawData, &asAny); err == nil {
			switch v := asAny.(type) {
			case string:
				_ = json.Unmarshal([]byte(v), &nodeData)
			case map[string]any:
				nodeData = v
			}
		}
	}

	rows, err := s.pool.Query(ctx, `
		SELECT status, "outputJson" FROM executions
		WHERE "executionId" = $1 AND "nodeId" = $2
	`, executionID, execNodeID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	outputData := make([]map[string]any, 0)
	for rows.Next() {
		var status models.ExecutionStatus
		var out []byte
		if err := rows.Scan(&status, &out); err != nil {
			return nil, err
		}
		item := map[string]any{"status": status}
		if len(out) > 0 {
			var parsed any
			if err := json.Unmarshal(out, &parsed); err == nil {
				item["outputJson"] = parsed
			} else {
				item["outputJson"] = string(out)
			}
		}
		outputData = append(outputData, item)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	nodeData["outputData"] = outputData
	return nodeData, nil
}
