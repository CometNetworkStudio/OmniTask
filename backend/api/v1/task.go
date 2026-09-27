package v1

import "time"

type TaskCreateRequest struct {
	Platform     string            `json:"platform" binding:"required"`
	Action       string            `json:"action" binding:"required"`
	CredentialID string            `json:"credential_id"`
	Params       map[string]string `json:"params"`
}

type TaskResponse struct {
	ID           string            `json:"id"`
	Platform     string            `json:"platform"`
	Action       string            `json:"action"`
	CredentialID string            `json:"credential_id,omitempty"`
	Params       map[string]string `json:"params"`
	Status       string            `json:"status"`
	Progress     int               `json:"progress"`
	Result       map[string]string `json:"result"`
	Error        string            `json:"error"`
	StartedAt    *time.Time        `json:"started_at"`
	FinishedAt   *time.Time        `json:"finished_at"`
	CreatedAt    time.Time         `json:"created_at"`
	UpdatedAt    time.Time         `json:"updated_at"`
}
