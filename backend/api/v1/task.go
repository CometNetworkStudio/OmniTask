package v1

import "time"

type TaskCreateRequest struct {
	Platform     string            `json:"platform" binding:"required"`
	Action       string            `json:"action" binding:"required"`
	CredentialID string            `json:"credential_id"`
	Params       map[string]string `json:"params"`
}

type TaskEventResponse struct {
	ID         string    `json:"id"`
	TaskID     string    `json:"task_id"`
	Type       string    `json:"type"`
	Kind       string    `json:"kind"`
	Percent    int       `json:"percent"`
	Message    string    `json:"message"`
	Level      string    `json:"level"`
	Step       string    `json:"step"`
	Label      string    `json:"label"`
	StepStatus string    `json:"step_status"`
	Current    int       `json:"current"`
	Total      int       `json:"total"`
	Unit       string    `json:"unit"`
	Stage      string    `json:"stage"`
	Key        string    `json:"key"`
	Value      string    `json:"value"`
	CreatedAt  time.Time `json:"created_at"`
}

type TaskResponse struct {
	ID           string            `json:"id"`
	Platform     string            `json:"platform"`
	Action       string            `json:"action"`
	CredentialID string            `json:"credential_id,omitempty"`
	Params       map[string]string `json:"params"`
	Status       string            `json:"status"`
	Progress     int               `json:"progress"`
	Result       any               `json:"result"`
	Error        string            `json:"error"`
	StartedAt    *time.Time        `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
