package v1

import "time"

type CredentialUpsertRequest struct {
	Platform string            `json:"platform" binding:"required"`
	Data     map[string]string `json:"data" binding:"required"`
}

type CredentialResponse struct {
	ID        string    `json:"id"`
	Platform  string    `json:"platform"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
}
