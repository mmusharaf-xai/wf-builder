package handlers

import (
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5"

	"github.com/wf-builder/workflow-builder-backend/internal/response"
	"github.com/wf-builder/workflow-builder-backend/internal/services"
)

type API struct {
	Workflows  *services.WorkflowService
	Nodes      *services.NodeService
	Executions *services.ExecutionService
	Webhooks   *services.WebhookEngine
}

func (a *API) Routes() http.Handler {
	r := chi.NewRouter()

	r.Get("/health", func(w http.ResponseWriter, r *http.Request) {
		response.OK(w, map[string]any{"status": "ok"})
	})

	r.Route("/api/workflows", func(r chi.Router) {
		r.Get("/", a.ListWorkflows)
		r.Post("/", a.CreateWorkflow)

		r.Route("/{id}", func(r chi.Router) {
			r.Get("/", a.GetWorkflow)
			r.Put("/", a.UpdateWorkflow)
			r.Delete("/", a.DeleteWorkflow)

			r.Post("/addNode", a.AddNode)
			r.Delete("/deleteNode", a.DeleteNode)
			r.Put("/updateNodeAndEdges", a.UpdateNodeAndEdges)
			r.Put("/updateNodeData", a.UpdateNodeData)
			r.Get("/getNodeData", a.GetNodeData)

			r.Get("/executions", a.ListExecutions)
			r.Get("/executions/{executionId}", a.GetExecution)
			r.Get("/executions/{executionId}/getNodeData", a.GetExecutionNodeData)
		})
	})

	// Webhook trigger — any method
	r.HandleFunc("/api/webhooks/{id}", a.HandleWebhook)

	return r
}

func (a *API) ListWorkflows(w http.ResponseWriter, r *http.Request) {
	q := r.URL.Query()
	page, _ := strconv.Atoi(q.Get("page"))
	limit, _ := strconv.Atoi(q.Get("limit"))
	result, err := a.Workflows.List(r.Context(), services.ListWorkflowsParams{
		Page:      page,
		Limit:     limit,
		Search:    q.Get("search"),
		SortOrder: q.Get("sortOrder"),
	})
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to fetch workflows")
		return
	}
	response.JSON(w, http.StatusOK, result)
}

func (a *API) CreateWorkflow(w http.ResponseWriter, r *http.Request) {
	var body struct {
		Name        string `json:"name"`
		Description string `json:"description"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	if body.Name == "" || body.Description == "" {
		response.Error(w, http.StatusBadRequest, "Name and description are required")
		return
	}
	wf, err := a.Workflows.Create(r.Context(), body.Name, body.Description)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to create workflow")
		return
	}
	response.Created(w, map[string]any{
		"message":  "workflow created",
		"workflow": wf,
	})
}

func (a *API) UpdateWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body map[string]any
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	delete(body, "is_deleted")
	delete(body, "id")
	delete(body, "user_id")

	wf, err := a.Workflows.Update(r.Context(), id, body)
	if errors.Is(err, pgx.ErrNoRows) || wf == nil {
		response.Error(w, http.StatusNotFound, "Workflow not found or update failed")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to update workflow")
		return
	}
	response.OK(w, map[string]any{
		"message":  "Workflow updated successfully",
		"workflow": wf,
	})
}

func (a *API) DeleteWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	wf, err := a.Workflows.SoftDelete(r.Context(), id)
	if errors.Is(err, pgx.ErrNoRows) || wf == nil {
		response.Error(w, http.StatusNotFound, "Workflow not found or delete failed")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete workflow")
		return
	}
	response.OK(w, map[string]any{
		"message":  "Workflow Deleted successfully",
		"workflow": wf,
	})
}

func (a *API) GetWorkflow(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID is required")
		return
	}
	wf, nodes, edges, err := a.Workflows.GetWithGraph(r.Context(), id)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	if wf == nil {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	response.OK(w, map[string]any{
		"workflow": wf,
		"nodes":    nodes,
		"edges":    edges,
	})
}

func (a *API) AddNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var in services.AddNodeInput
	if err := json.NewDecoder(r.Body).Decode(&in); err != nil {
		response.Error(w, http.StatusBadRequest, "Invalid request body")
		return
	}
	node, err := a.Nodes.Add(r.Context(), id, in)
	if errors.Is(err, services.ErrInvalidInput) {
		response.Error(w, http.StatusBadRequest, "Missing required parameters")
		return
	}
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if errors.Is(err, services.ErrConflict) {
		response.Error(w, http.StatusBadRequest, "Webhook path already exists")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	response.Created(w, map[string]any{
		"message": "Node added successfully",
		"node":    node,
	})
}

func (a *API) DeleteNode(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		NodeID string `json:"nodeId"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NodeID == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID and Node ID are required")
		return
	}
	node, err := a.Nodes.Delete(r.Context(), id, body.NodeID)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Node not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to delete node")
		return
	}
	response.OK(w, map[string]any{
		"message": "Node deleted successfully",
		"node":    node,
	})
}

func (a *API) UpdateNodeAndEdges(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		Nodes []services.GraphNodeInput `json:"nodes"`
		Edges []services.GraphEdgeInput `json:"edges"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil {
		response.Error(w, http.StatusBadRequest, "Workflow ID, Nodes and Edges are required")
		return
	}
	if body.Nodes == nil || body.Edges == nil {
		response.Error(w, http.StatusBadRequest, "Workflow ID, Nodes and Edges are required")
		return
	}
	err := a.Nodes.UpdateNodesAndEdges(r.Context(), id, body.Nodes, body.Edges)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	response.OK(w, map[string]any{"message": "Workflow updated"})
}

func (a *API) UpdateNodeData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	var body struct {
		NodeID string `json:"nodeId"`
		Data   any    `json:"data"`
	}
	if err := json.NewDecoder(r.Body).Decode(&body); err != nil || body.NodeID == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID and Node ID are required")
		return
	}
	node, err := a.Nodes.UpdateNodeData(r.Context(), id, body.NodeID, body.Data)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Node not found")
		return
	}
	if errors.Is(err, services.ErrConflict) {
		response.Error(w, http.StatusBadRequest, "Webhook path and method combination already exists in another workflow")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	response.OK(w, map[string]any{"node": node})
}

func (a *API) GetNodeData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	nodeID := r.URL.Query().Get("nodeId")
	if id == "" || nodeID == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID and Node ID are required")
		return
	}
	data, err := a.Nodes.GetNodeData(r.Context(), id, nodeID)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Internal server error")
		return
	}
	response.OK(w, map[string]any{"data": data})
}

func (a *API) ListExecutions(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID is required")
		return
	}
	page, _ := strconv.Atoi(r.URL.Query().Get("page_number"))
	size, _ := strconv.Atoi(r.URL.Query().Get("page_size"))

	filter := services.ListExecutionsFilter{}
	if raw := r.URL.Query().Get("from"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			// also accept date-only YYYY-MM-DD
			t, err = time.Parse("2006-01-02", raw)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "Invalid 'from' datetime (use RFC3339 or YYYY-MM-DD)")
				return
			}
		}
		filter.From = &t
	}
	if raw := r.URL.Query().Get("to"); raw != "" {
		t, err := time.Parse(time.RFC3339, raw)
		if err != nil {
			t, err = time.Parse("2006-01-02", raw)
			if err != nil {
				response.Error(w, http.StatusBadRequest, "Invalid 'to' datetime (use RFC3339 or YYYY-MM-DD)")
				return
			}
			// inclusive end of day for date-only
			end := t.Add(24*time.Hour - time.Nanosecond)
			t = end
		}
		filter.To = &t
	}

	result, err := a.Executions.List(r.Context(), id, page, size, filter)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Failed to get workflow history")
		return
	}
	response.OK(w, map[string]any{"data": result})
}

func (a *API) GetExecution(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	executionID := chi.URLParam(r, "executionId")
	if id == "" || executionID == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID and ExecutionId is required")
		return
	}
	wf, nodes, edges, err := a.Executions.GetGraph(r.Context(), id, executionID)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Workflow not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	response.OK(w, map[string]any{
		"workflow": wf,
		"nodes":    nodes,
		"edges":    edges,
	})
}

func (a *API) GetExecutionNodeData(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	executionID := chi.URLParam(r, "executionId")
	nodeID := r.URL.Query().Get("nodeId")
	if id == "" || executionID == "" {
		response.Error(w, http.StatusBadRequest, "Workflow ID and ExecutionId is required")
		return
	}
	data, err := a.Executions.GetNodeData(r.Context(), id, executionID, nodeID)
	if errors.Is(err, services.ErrNotFound) {
		response.Error(w, http.StatusNotFound, "Node not found")
		return
	}
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Something went wrong")
		return
	}
	response.OK(w, map[string]any{"nodeData": data})
}

func (a *API) HandleWebhook(w http.ResponseWriter, r *http.Request) {
	id := chi.URLParam(r, "id")
	if id == "" {
		response.Error(w, http.StatusBadRequest, "Webhook ID is required")
		return
	}
	result, err := a.Webhooks.Handle(r.Context(), id, r.Method, r)
	if err != nil {
		response.Error(w, http.StatusInternalServerError, "Webhook processing failed")
		return
	}
	for k, v := range result.Headers {
		w.Header().Set(k, v)
	}
	status := result.Status
	if status == 0 {
		status = http.StatusOK
	}
	response.JSON(w, status, result.Body)
}
