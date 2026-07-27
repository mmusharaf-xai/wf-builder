package models

import (
	"encoding/json"
	"time"
)

type NodeType string

const (
	NodeTypeWebhook         NodeType = "WEBHOOK_NODE"
	NodeTypeCode            NodeType = "CODE_NODE"
	NodeTypeWebhookResponse NodeType = "WEBHOOK_RESPONSE_NODE"
)

type WebhookMethod string

const (
	MethodGET    WebhookMethod = "GET"
	MethodPOST   WebhookMethod = "POST"
	MethodPUT    WebhookMethod = "PUT"
	MethodDELETE WebhookMethod = "DELETE"
)

type ExecutionStatus string

const (
	StatusPending   ExecutionStatus = "PENDING"
	StatusCompleted ExecutionStatus = "COMPLETED"
	StatusFailed    ExecutionStatus = "FAILED"
)

type Workflow struct {
	ID          string    `json:"id"`
	IsActive    bool      `json:"is_active"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	UserID      string    `json:"user_id"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
	IsDeleted   bool      `json:"is_deleted"`
}

type WorkflowNode struct {
	ID          string          `json:"id"`
	WorkflowID  string          `json:"workflowId"`
	Type        NodeType        `json:"type"`
	PositionX   float64         `json:"positionX"`
	PositionY   float64         `json:"positionY"`
	Label       string          `json:"label"`
	Icon        *string         `json:"icon,omitempty"`
	Color       *string         `json:"color,omitempty"`
	Description *string         `json:"description,omitempty"`
	Data        json.RawMessage `json:"data,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
}

type WorkflowEdge struct {
	ID         string    `json:"id"`
	WorkflowID string    `json:"workflowId"`
	Source     string    `json:"source"`
	Target     string    `json:"target"`
	CreatedAt  time.Time `json:"createdAt"`
	UpdatedAt  time.Time `json:"updatedAt"`
}

type Webhook struct {
	ID         string        `json:"id"`
	Path       string        `json:"path"`
	WorkflowID string        `json:"workflowId"`
	Method     WebhookMethod `json:"method"`
	NodeID     string        `json:"nodeId"`
	CreatedAt  time.Time     `json:"createdAt"`
	UpdatedAt  time.Time     `json:"updatedAt"`
}

type ExecutionHistory struct {
	ID          string          `json:"id"`
	WorkflowID  string          `json:"workflowId"`
	Status      ExecutionStatus `json:"status"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	CompletedAt *time.Time      `json:"completedAt"`
}

type ExecutionNode struct {
	ID             string          `json:"id"`
	Type           NodeType        `json:"type"`
	PositionX      float64         `json:"positionX"`
	PositionY      float64         `json:"positionY"`
	Label          string          `json:"label"`
	Icon           *string         `json:"icon,omitempty"`
	Color          *string         `json:"color,omitempty"`
	Description    *string         `json:"description,omitempty"`
	Data           json.RawMessage `json:"data,omitempty"`
	WorkflowNodeID string          `json:"workflowNodeId"`
	ExecutionID    string          `json:"executionId"`
	CreatedAt      time.Time       `json:"createdAt"`
	UpdatedAt      time.Time       `json:"updatedAt"`
}

type ExecutionEdge struct {
	ID          string    `json:"id"`
	Source      string    `json:"source"`
	Target      string    `json:"target"`
	ExecutionID string    `json:"executionId"`
	CreatedAt   time.Time `json:"createdAt"`
	UpdatedAt   time.Time `json:"updatedAt"`
}

type Execution struct {
	ID          string          `json:"id"`
	ExecutionID string          `json:"executionId"`
	NodeID      string          `json:"nodeId"`
	Status      ExecutionStatus `json:"status"`
	OutputJSON  json.RawMessage `json:"outputJson,omitempty"`
	CreatedAt   time.Time       `json:"createdAt"`
	UpdatedAt   time.Time       `json:"updatedAt"`
	CompletedAt *time.Time      `json:"completedAt"`
}

// API graph node shape returned to the frontend.
type GraphNode struct {
	ID             string         `json:"id"`
	Type           NodeType       `json:"type"`
	Position       Position       `json:"position"`
	WorkflowNodeID string         `json:"workflowNodeId,omitempty"`
	Data           map[string]any `json:"data"`
}

type Position struct {
	X float64 `json:"x"`
	Y float64 `json:"y"`
}

type GraphEdge struct {
	ID     string `json:"id"`
	Source string `json:"source"`
	Target string `json:"target"`
}

// NodeData payloads (stored as JSON on workflowNodes.data).

type WebhookNodeData struct {
	Parameters struct {
		Method      string `json:"method"`
		Path        string `json:"path"`
		RespondType string `json:"respondType"`
	} `json:"parameters"`
	Settings struct {
		AllowMultipleHTTPS bool   `json:"allowMultipleHttps"`
		Notes              string `json:"notes"`
	} `json:"settings"`
}

type CodeNodeData struct {
	Parameters struct {
		Code string `json:"code"`
		Type string `json:"type"`
	} `json:"parameters"`
	Settings struct {
		RetryOnFail struct {
			IsEnabled         bool `json:"isEnabled"`
			MaxTries          int  `json:"maxTries"`
			WaitBetweenTries  int  `json:"waitBetweenTries"`
		} `json:"retryOnFail"`
		OnError string `json:"onError"`
		Notes   string `json:"notes"`
		Timeout int    `json:"timeout"`
	} `json:"settings"`
}

type WebhookResponseNodeData struct {
	Parameters struct {
		RespondWith     string              `json:"respondWith"`
		ResponseValue   string              `json:"responseValue"`
		ResponseCode    int                 `json:"responseCode"`
		ResponseHeaders []map[string]string `json:"responseHeaders"`
	} `json:"parameters"`
	Settings struct {
		RetryOnFail struct {
			IsEnabled        bool `json:"isEnabled"`
			MaxTries         int  `json:"maxTries"`
			WaitBetweenTries int  `json:"waitBetweenTries"`
		} `json:"retryOnFail"`
		OnError string `json:"onError"`
		Notes   string `json:"notes"`
	} `json:"settings"`
}
