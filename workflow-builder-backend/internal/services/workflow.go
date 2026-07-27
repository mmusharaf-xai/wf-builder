package services

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wf-builder/workflow-builder-backend/internal/models"
)

type WorkflowService struct {
	pool   *pgxpool.Pool
	userID string
}

func NewWorkflowService(pool *pgxpool.Pool, userID string) *WorkflowService {
	return &WorkflowService{pool: pool, userID: userID}
}

type ListWorkflowsParams struct {
	Page      int
	Limit     int
	Search    string
	SortOrder string
}

type ListWorkflowsResult struct {
	Workflows []models.Workflow `json:"workflows"`
	Metadata  map[string]any    `json:"metadata"`
	Error     string            `json:"error,omitempty"`
}

func (s *WorkflowService) List(ctx context.Context, p ListWorkflowsParams) (*ListWorkflowsResult, error) {
	if p.Limit < 1 {
		p.Limit = 25
	}
	if p.Limit > 100 {
		p.Limit = 100
	}
	if p.Page < 1 {
		p.Page = 1
	}
	sortOrder := "DESC"
	if strings.EqualFold(p.SortOrder, "asc") {
		sortOrder = "ASC"
	}
	offset := (p.Page - 1) * p.Limit

	where := `user_id = $1 AND is_deleted = false`
	args := []any{s.userID}
	argN := 2
	search := strings.TrimSpace(p.Search)
	if search != "" {
		// Case-insensitive match on name or description (backend search for worklist).
		where += fmt.Sprintf(` AND (name ILIKE $%d OR description ILIKE $%d)`, argN, argN)
		args = append(args, "%"+search+"%")
		argN++
	}

	var total int
	countQ := fmt.Sprintf(`SELECT COUNT(*) FROM workflows WHERE %s`, where)
	if err := s.pool.QueryRow(ctx, countQ, args...).Scan(&total); err != nil {
		return nil, err
	}

	query := fmt.Sprintf(`
		SELECT id, is_active, name, description, user_id, "createdAt", "updatedAt", is_deleted
		FROM workflows
		WHERE %s
		ORDER BY "updatedAt" %s, id %s
		LIMIT $%d OFFSET $%d
	`, where, sortOrder, sortOrder, argN, argN+1)
	args = append(args, p.Limit, offset)

	rows, err := s.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	workflows := make([]models.Workflow, 0)
	for rows.Next() {
		var w models.Workflow
		if err := rows.Scan(&w.ID, &w.IsActive, &w.Name, &w.Description, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.IsDeleted); err != nil {
			return nil, err
		}
		workflows = append(workflows, w)
	}
	if err := rows.Err(); err != nil {
		return nil, err
	}

	totalPages := 0
	if p.Limit > 0 {
		totalPages = (total + p.Limit - 1) / p.Limit
	}

	return &ListWorkflowsResult{
		Workflows: workflows,
		Metadata: map[string]any{
			"total":           total,
			"page":            p.Page,
			"totalPages":      totalPages,
			"hasNextPage":     p.Page < totalPages,
			"hasPreviousPage": p.Page > 1,
			"limit":           p.Limit,
		},
	}, nil
}

func (s *WorkflowService) Create(ctx context.Context, name, description string) (*models.Workflow, error) {
	now := time.Now().UTC()
	w := models.Workflow{
		ID:          uuid.NewString(),
		Name:        name,
		Description: description,
		UserID:      s.userID,
		CreatedAt:   now,
		UpdatedAt:   now,
	}
	_, err := s.pool.Exec(ctx, `
		INSERT INTO workflows (id, is_active, name, description, user_id, "createdAt", "updatedAt", is_deleted)
		VALUES ($1, false, $2, $3, $4, $5, $6, false)
	`, w.ID, w.Name, w.Description, w.UserID, w.CreatedAt, w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *WorkflowService) Update(ctx context.Context, id string, fields map[string]any) (*models.Workflow, error) {
	w, err := s.GetOwned(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, pgx.ErrNoRows
	}

	if name, ok := fields["name"].(string); ok && name != "" {
		w.Name = name
	}
	if desc, ok := fields["description"].(string); ok {
		w.Description = desc
	}
	if active, ok := fields["is_active"].(bool); ok {
		w.IsActive = active
	}
	w.UpdatedAt = time.Now().UTC()

	_, err = s.pool.Exec(ctx, `
		UPDATE workflows SET name = $2, description = $3, is_active = $4, "updatedAt" = $5
		WHERE id = $1
	`, w.ID, w.Name, w.Description, w.IsActive, w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WorkflowService) SoftDelete(ctx context.Context, id string) (*models.Workflow, error) {
	w, err := s.GetOwned(ctx, id)
	if err != nil {
		return nil, err
	}
	if w == nil {
		return nil, pgx.ErrNoRows
	}
	w.IsDeleted = true
	w.UpdatedAt = time.Now().UTC()
	_, err = s.pool.Exec(ctx, `
		UPDATE workflows SET is_deleted = true, "updatedAt" = $2 WHERE id = $1
	`, id, w.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return w, nil
}

func (s *WorkflowService) GetOwned(ctx context.Context, id string) (*models.Workflow, error) {
	var w models.Workflow
	err := s.pool.QueryRow(ctx, `
		SELECT id, is_active, name, description, user_id, "createdAt", "updatedAt", is_deleted
		FROM workflows
		WHERE id = $1 AND user_id = $2 AND is_deleted = false
	`, id, s.userID).Scan(
		&w.ID, &w.IsActive, &w.Name, &w.Description, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.IsDeleted,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *WorkflowService) GetByID(ctx context.Context, id string) (*models.Workflow, error) {
	var w models.Workflow
	err := s.pool.QueryRow(ctx, `
		SELECT id, is_active, name, description, user_id, "createdAt", "updatedAt", is_deleted
		FROM workflows WHERE id = $1
	`, id).Scan(
		&w.ID, &w.IsActive, &w.Name, &w.Description, &w.UserID, &w.CreatedAt, &w.UpdatedAt, &w.IsDeleted,
	)
	if err == pgx.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, err
	}
	return &w, nil
}

func (s *WorkflowService) GetWithGraph(ctx context.Context, id string) (*models.Workflow, []models.GraphNode, []models.WorkflowEdge, error) {
	w, err := s.GetOwned(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	if w == nil {
		return nil, nil, nil, nil
	}

	rows, err := s.pool.Query(ctx, `
		SELECT id, "workflowId", type, "positionX", "positionY", label, icon, color, description, "createdAt", "updatedAt"
		FROM "workflowNodes" WHERE "workflowId" = $1
	`, id)
	if err != nil {
		return nil, nil, nil, err
	}
	defer rows.Close()

	nodes := make([]models.GraphNode, 0)
	for rows.Next() {
		var (
			nid, workflowID, label string
			nodeType               models.NodeType
			px, py                 float64
			icon, color, desc      *string
			createdAt, updatedAt   time.Time
		)
		if err := rows.Scan(&nid, &workflowID, &nodeType, &px, &py, &label, &icon, &color, &desc, &createdAt, &updatedAt); err != nil {
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
		nodes = append(nodes, models.GraphNode{
			ID:       nid,
			Type:     nodeType,
			Position: models.Position{X: px, Y: py},
			Data:     data,
		})
	}
	if err := rows.Err(); err != nil {
		return nil, nil, nil, err
	}

	edges, err := s.listEdges(ctx, id)
	if err != nil {
		return nil, nil, nil, err
	}
	return w, nodes, edges, nil
}

func (s *WorkflowService) listEdges(ctx context.Context, workflowID string) ([]models.WorkflowEdge, error) {
	rows, err := s.pool.Query(ctx, `
		SELECT id, "workflowId", source, target, "createdAt", "updatedAt"
		FROM "workflowEdges" WHERE "workflowId" = $1
	`, workflowID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	edges := make([]models.WorkflowEdge, 0)
	for rows.Next() {
		var e models.WorkflowEdge
		if err := rows.Scan(&e.ID, &e.WorkflowID, &e.Source, &e.Target, &e.CreatedAt, &e.UpdatedAt); err != nil {
			return nil, err
		}
		edges = append(edges, e)
	}
	return edges, rows.Err()
}
