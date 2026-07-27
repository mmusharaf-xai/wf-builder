package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/wf-builder/workflow-builder-backend/internal/models"
)

type WebhookEngine struct {
	pool   *pgxpool.Pool
	parser *ParseValuesService
}

func NewWebhookEngine(pool *pgxpool.Pool, parser *ParseValuesService) *WebhookEngine {
	return &WebhookEngine{pool: pool, parser: parser}
}

type WebhookResult struct {
	Status  int
	Headers map[string]string
	Body    map[string]any
}

func (e *WebhookEngine) Handle(ctx context.Context, path, method string, r *http.Request) (*WebhookResult, error) {
	var (
		webhookID, nodeID, workflowID string
		webhookMethod                 string
		nodeDataRaw                   []byte
	)

	err := e.pool.QueryRow(ctx, `
		SELECT w.id, w."nodeId", w."workflowId", w.method, n.data
		FROM webhooks w
		JOIN "workflowNodes" n ON n.id = w."nodeId"
		WHERE w.path = $1
		LIMIT 1
	`, path).Scan(&webhookID, &nodeID, &workflowID, &webhookMethod, &nodeDataRaw)
	if errors.Is(err, pgx.ErrNoRows) {
		return &WebhookResult{
			Status: http.StatusBadRequest,
			Body:   map[string]any{"error": true, "message": "Webhook node not found"},
		}, nil
	}
	if err != nil {
		return nil, err
	}

	if webhookMethod != method {
		return &WebhookResult{
			Status: http.StatusMethodNotAllowed,
			Body:   map[string]any{"error": true, "message": "Invalid method."},
		}, nil
	}

	var webhookNodeData models.WebhookNodeData
	_ = unmarshalFlexible(nodeDataRaw, &webhookNodeData)

	now := time.Now().UTC()
	executionID := uuid.NewString()
	_, err = e.pool.Exec(ctx, `
		INSERT INTO "executionsHistory" (id, "workflowId", status, "createdAt", "updatedAt")
		VALUES ($1, $2, 'PENDING', $3, $4)
	`, executionID, workflowID, now, now)
	if err != nil {
		return nil, err
	}

	// Load workflow graph
	nodeRows, err := e.pool.Query(ctx, `
		SELECT id, type, "positionX", "positionY", label, icon, color, description, data
		FROM "workflowNodes" WHERE "workflowId" = $1
	`, workflowID)
	if err != nil {
		return nil, err
	}

	type wfNode struct {
		ID          string
		Type        models.NodeType
		PositionX   float64
		PositionY   float64
		Label       string
		Icon        *string
		Color       *string
		Description *string
		Data        []byte
	}
	workflowNodes := make([]wfNode, 0)
	for nodeRows.Next() {
		var n wfNode
		if err := nodeRows.Scan(&n.ID, &n.Type, &n.PositionX, &n.PositionY, &n.Label, &n.Icon, &n.Color, &n.Description, &n.Data); err != nil {
			nodeRows.Close()
			return nil, err
		}
		workflowNodes = append(workflowNodes, n)
	}
	nodeRows.Close()
	if err := nodeRows.Err(); err != nil {
		return nil, err
	}

	edgeRows, err := e.pool.Query(ctx, `
		SELECT id, source, target FROM "workflowEdges" WHERE "workflowId" = $1
	`, workflowID)
	if err != nil {
		return nil, err
	}
	type wfEdge struct {
		ID, Source, Target string
	}
	workflowEdges := make([]wfEdge, 0)
	for edgeRows.Next() {
		var ed wfEdge
		if err := edgeRows.Scan(&ed.ID, &ed.Source, &ed.Target); err != nil {
			edgeRows.Close()
			return nil, err
		}
		workflowEdges = append(workflowEdges, ed)
	}
	edgeRows.Close()
	if err := edgeRows.Err(); err != nil {
		return nil, err
	}

	// Snapshot into execution nodes/edges
	nodeMapping := map[string]string{} // workflowNodeId -> executionNodeId
	executionNodes := map[string]*models.ExecutionNode{}
	var executionNodeList []*models.ExecutionNode

	for _, n := range workflowNodes {
		execNodeID := uuid.NewString()
		nodeMapping[n.ID] = execNodeID
		dataJSON := "null"
		if len(n.Data) > 0 {
			dataJSON = string(n.Data)
		}
		_, err := e.pool.Exec(ctx, `
			INSERT INTO "executionNodes"
				(id, type, "positionX", "positionY", label, icon, color, description, data, "createdAt", "updatedAt", "workflowNodeId", "executionId")
			VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9::jsonb,$10,$11,$12,$13)
		`, execNodeID, n.Type, n.PositionX, n.PositionY, n.Label, n.Icon, n.Color, n.Description, dataJSON, now, now, n.ID, executionID)
		if err != nil {
			_ = e.failHistory(ctx, executionID)
			return nil, err
		}
		en := &models.ExecutionNode{
			ID:             execNodeID,
			Type:           n.Type,
			PositionX:      n.PositionX,
			PositionY:      n.PositionY,
			Label:          n.Label,
			Icon:           n.Icon,
			Color:          n.Color,
			Description:    n.Description,
			Data:           n.Data,
			WorkflowNodeID: n.ID,
			ExecutionID:    executionID,
		}
		executionNodes[execNodeID] = en
		executionNodeList = append(executionNodeList, en)
	}

	executionEdges := make([]execEdge, 0, len(workflowEdges))
	for _, ed := range workflowEdges {
		src := nodeMapping[ed.Source]
		tgt := nodeMapping[ed.Target]
		if src == "" || tgt == "" {
			_ = e.failHistory(ctx, executionID)
			return nil, fmt.Errorf("missing node mapping for edge %s", ed.ID)
		}
		eid := uuid.NewString()
		_, err := e.pool.Exec(ctx, `
			INSERT INTO "executionEdges" (id, source, target, "createdAt", "updatedAt", "executionId")
			VALUES ($1,$2,$3,$4,$5,$6)
		`, eid, src, tgt, now, now, executionID)
		if err != nil {
			_ = e.failHistory(ctx, executionID)
			return nil, err
		}
		executionEdges = append(executionEdges, execEdge{ID: eid, Source: src, Target: tgt})
	}

	webhookExecNodeID := nodeMapping[nodeID]
	if webhookExecNodeID == "" {
		_ = e.failHistory(ctx, executionID)
		return &WebhookResult{
			Status: http.StatusInternalServerError,
			Body:   map[string]any{"error": true, "message": "Webhook node not found in execution flow"},
		}, nil
	}

	edgePairs := make([]struct{ Source, Target string }, 0, len(executionEdges))
	for _, ed := range executionEdges {
		edgePairs = append(edgePairs, struct{ Source, Target string }{ed.Source, ed.Target})
	}
	validNodeIDs := getValidConnectedNodes(webhookExecNodeID, edgePairs)

	// Request body + headers + params as webhook output
	bodyBytes, _ := io.ReadAll(io.LimitReader(r.Body, 10<<20))
	var body any
	if len(bodyBytes) > 0 {
		if err := json.Unmarshal(bodyBytes, &body); err != nil {
			body = string(bodyBytes)
		}
	} else {
		body = map[string]any{}
	}
	headers := map[string]string{}
	for k, vals := range r.Header {
		if len(vals) > 0 {
			headers[k] = vals[0]
		}
	}
	params := map[string]string{}
	for k, vals := range r.URL.Query() {
		if len(vals) > 0 {
			params[k] = vals[0]
		}
	}
	output := map[string]any{
		"headers": headers,
		"body":    body,
		"params":  params,
	}
	outBytes, _ := json.Marshal(output)
	webhookExecutionID := uuid.NewString()
	completedAt := time.Now().UTC()
	_, err = e.pool.Exec(ctx, `
		INSERT INTO executions (id, "executionId", "nodeId", "createdAt", "updatedAt", status, "outputJson", "completedAt")
		VALUES ($1,$2,$3,$4,$5,'COMPLETED',$6::jsonb,$7)
	`, webhookExecutionID, executionID, webhookExecNodeID, completedAt, completedAt, string(outBytes), completedAt)
	if err != nil {
		_ = e.failHistory(ctx, executionID)
		return nil, err
	}

	// n8n responseMode "onReceived" (== IMMEDIATELY):
	// - HTTP response is returned as soon as the webhook node finishes
	// - workflow still continues for any downstream nodes
	// - when there are no further nodes, the run is marked success immediately
	//   (see extractWebhookOnReceivedResponse + workflow completion lifecycle)
	if webhookNodeData.Parameters.RespondType == "IMMEDIATELY" {
		immediate := &WebhookResult{
			Status: http.StatusOK,
			Body: map[string]any{
				"error":   false,
				// n8n default body when no custom responseData is set
				"message": "Workflow was started",
				"data":    map[string]any{"executionId": executionID},
			},
		}

		if !hasValidDownstream(webhookExecNodeID, executionEdges, validNodeIDs) {
			// Single webhook (or no connected next nodes): mark run COMPLETED.
			// Previously we returned while executionsHistory stayed PENDING forever.
			if err := e.completeHistory(ctx, executionID); err != nil {
				return nil, err
			}
			return immediate, nil
		}

		// Downstream nodes exist: respond now, finish the graph in the background
		// so the HTTP client is not blocked (n8n onReceived behavior).
		go func() {
			bg, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
			defer cancel()
			res := e.runDownstream(
				bg,
				executionID,
				webhookExecNodeID,
				executionNodes,
				executionEdges,
				validNodeIDs,
			)
			if res.hardFailed || res.err != nil {
				_ = e.failHistory(bg, executionID)
				return
			}
			_ = e.completeHistory(bg, executionID)
		}()
		return immediate, nil
	}

	// LAST_NODE / RESPONSE_WEBHOOK: walk the chain, then respond.
	runResult := e.runDownstream(
		ctx,
		executionID,
		webhookExecNodeID,
		executionNodes,
		executionEdges,
		validNodeIDs,
	)
	lastOutput := runResult.lastOutput
	lastWebhookResponseNode := runResult.lastWebhookResponseNode
	lastWebhookResponseData := runResult.lastWebhookResponseData

	if runResult.hardFailed {
		_ = e.failHistory(ctx, executionID)
		return &WebhookResult{
			Status: http.StatusOK,
			Body: map[string]any{
				"error":   true,
				"message": "Workflow failed",
				"data":    map[string]any{"error": runResult.failMessage},
			},
		}, nil
	}
	if runResult.err != nil {
		_ = e.failHistory(ctx, executionID)
		return nil, runResult.err
	}

	if err := e.completeHistory(ctx, executionID); err != nil {
		return nil, err
	}

	if webhookNodeData.Parameters.RespondType == "RESPONSE_WEBHOOK" && lastWebhookResponseNode != nil && lastWebhookResponseData != nil {
		headersOut := map[string]string{}
		for _, h := range lastWebhookResponseData.Parameters.ResponseHeaders {
			label := h["label"]
			value := h["value"]
			if label != "" {
				headersOut[label] = value
			}
		}
		status := lastWebhookResponseData.Parameters.ResponseCode
		if status == 0 {
			status = 200
		}
		// Prefer parsed value from lastOutput if available
		data := lastWebhookResponseData.Parameters.ResponseValue
		if m, ok := lastOutput.(map[string]any); ok {
			if v, ok := m["responseValue"]; ok {
				data = fmt.Sprint(v)
			}
		}
		var bodyData any
		if err := json.Unmarshal([]byte(data), &bodyData); err != nil {
			bodyData = data
		}
		return &WebhookResult{
			Status:  status,
			Headers: headersOut,
			Body: map[string]any{
				"error": false,
				"data":  bodyData,
			},
		}, nil
	}

	if webhookNodeData.Parameters.RespondType == "LAST_NODE" {
		return &WebhookResult{
			Status: http.StatusOK,
			Body: map[string]any{
				"error": false,
				"data":  lastOutput,
			},
		}, nil
	}

	return &WebhookResult{
		Status: http.StatusOK,
		Body: map[string]any{
			"error":   false,
			"message": "Workflow Completed",
			"data":    lastOutput,
		},
	}, nil
}

type execEdge struct {
	ID, Source, Target string
}

type downstreamResult struct {
	lastOutput              any
	lastWebhookResponseNode *models.ExecutionNode
	lastWebhookResponseData *models.WebhookResponseNodeData
	hardFailed              bool
	failMessage             string
	err                     error
}

// runDownstream walks nodes after the webhook trigger (linear chain).
// Returns hardFailed=true when a node fails with onError=STOP.
func (e *WebhookEngine) runDownstream(
	ctx context.Context,
	executionID string,
	startNodeID string,
	executionNodes map[string]*models.ExecutionNode,
	executionEdges []execEdge,
	validNodeIDs map[string]bool,
) downstreamResult {
	var result downstreamResult
	currentID := startNodeID

	for {
		var nextTarget string
		for _, ed := range executionEdges {
			if ed.Source == currentID {
				nextTarget = ed.Target
				break
			}
		}
		if nextTarget == "" || !validNodeIDs[nextTarget] {
			break
		}
		current := executionNodes[nextTarget]
		if current == nil {
			break
		}
		currentID = current.ID

		execRecordID := uuid.NewString()
		now := time.Now().UTC()
		if _, err := e.pool.Exec(ctx, `
			INSERT INTO executions (id, "executionId", "nodeId", "createdAt", "updatedAt", status)
			VALUES ($1,$2,$3,$4,$5,'PENDING')
		`, execRecordID, executionID, current.ID, now, now); err != nil {
			result.err = err
			return result
		}

		var outputJSON any
		var runErr error

		switch current.Type {
		case models.NodeTypeCode:
			var codeData models.CodeNodeData
			_ = unmarshalFlexible(current.Data, &codeData)
			code := codeData.Parameters.Code
			parsed, perr := e.parser.GetParsedValues(ctx, executionID, code)
			if perr != nil {
				runErr = perr
			} else {
				timeout := codeData.Settings.Timeout
				if timeout == 0 {
					timeout = 10000
				}
				res := RunJSScript(parsed, timeout)
				if res.Error != "" {
					runErr = errors.New(res.Error)
				} else {
					outputJSON = map[string]any{
						"logs":          res.Logs,
						"result":        res.Data,
						"executionTime": res.ExecutionTime,
					}
				}
			}

		case models.NodeTypeWebhookResponse:
			var respData models.WebhookResponseNodeData
			_ = unmarshalFlexible(current.Data, &respData)
			result.lastWebhookResponseNode = current
			result.lastWebhookResponseData = &respData
			parsed, perr := e.parser.GetParsedValues(ctx, executionID, respData.Parameters.ResponseValue)
			if perr != nil {
				runErr = perr
			} else {
				code := respData.Parameters.ResponseCode
				if code == 0 {
					code = 200
				}
				outputJSON = map[string]any{
					"responseValue":   parsed,
					"responseCode":    code,
					"responseHeaders": respData.Parameters.ResponseHeaders,
				}
			}

		default:
			outputJSON = map[string]any{
				"message": "Node processed",
				"type":    current.Type,
			}
		}

		if runErr != nil {
			errBytes, _ := json.Marshal(map[string]any{"error": runErr.Error()})
			failedAt := time.Now().UTC()
			_, _ = e.pool.Exec(ctx, `
				UPDATE executions SET status = 'FAILED', "outputJson" = $2::jsonb, "completedAt" = $3, "updatedAt" = $3
				WHERE id = $1
			`, execRecordID, string(errBytes), failedAt)

			onError := "CONTINUE"
			var settings struct {
				Settings struct {
					OnError string `json:"onError"`
				} `json:"settings"`
			}
			_ = unmarshalFlexible(current.Data, &settings)
			if settings.Settings.OnError != "" {
				onError = settings.Settings.OnError
			}
			if onError == "STOP" {
				result.hardFailed = true
				result.failMessage = runErr.Error()
				return result
			}
			continue
		}

		outB, _ := json.Marshal(outputJSON)
		doneAt := time.Now().UTC()
		if _, err := e.pool.Exec(ctx, `
			UPDATE executions SET status = 'COMPLETED', "outputJson" = $2::jsonb, "completedAt" = $3, "updatedAt" = $3
			WHERE id = $1
		`, execRecordID, string(outB), doneAt); err != nil {
			result.err = err
			return result
		}
		result.lastOutput = outputJSON
	}

	return result
}

func hasValidDownstream(fromID string, edges []execEdge, validNodeIDs map[string]bool) bool {
	for _, ed := range edges {
		if ed.Source == fromID && validNodeIDs[ed.Target] {
			return true
		}
	}
	return false
}

func (e *WebhookEngine) completeHistory(ctx context.Context, executionID string) error {
	now := time.Now().UTC()
	_, err := e.pool.Exec(ctx, `
		UPDATE "executionsHistory" SET status = 'COMPLETED', "completedAt" = $2, "updatedAt" = $2
		WHERE id = $1
	`, executionID, now)
	return err
}

func (e *WebhookEngine) failHistory(ctx context.Context, executionID string) error {
	now := time.Now().UTC()
	_, err := e.pool.Exec(ctx, `
		UPDATE "executionsHistory" SET status = 'FAILED', "completedAt" = $2, "updatedAt" = $2
		WHERE id = $1
	`, executionID, now)
	return err
}

func getValidConnectedNodes(webhookNodeID string, edges []struct{ Source, Target string }) map[string]bool {
	valid := map[string]bool{}
	incoming := map[string]bool{}
	outgoing := map[string]bool{}
	for _, e := range edges {
		outgoing[e.Source] = true
		incoming[e.Target] = true
	}
	valid[webhookNodeID] = true

	isValid := func(nodeID string) bool {
		if nodeID == webhookNodeID {
			return outgoing[nodeID]
		}
		if !outgoing[nodeID] {
			return incoming[nodeID]
		}
		return incoming[nodeID] && outgoing[nodeID]
	}

	var traverse func(string)
	traverse = func(current string) {
		for _, e := range edges {
			if e.Source != current {
				continue
			}
			if isValid(e.Target) && !valid[e.Target] {
				valid[e.Target] = true
				traverse(e.Target)
			}
		}
	}
	traverse(webhookNodeID)
	return valid
}

func unmarshalFlexible(raw []byte, dest any) error {
	if len(raw) == 0 {
		return nil
	}
	var asAny any
	if err := json.Unmarshal(raw, &asAny); err != nil {
		return err
	}
	if s, ok := asAny.(string); ok {
		return json.Unmarshal([]byte(s), dest)
	}
	b, err := json.Marshal(asAny)
	if err != nil {
		return err
	}
	return json.Unmarshal(b, dest)
}
