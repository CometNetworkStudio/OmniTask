package v1

import "time"

type ScriptInstallRequest struct {
	URL string `json:"url" binding:"required"`
}

type ScriptToggleRequest struct {
	Enabled bool `json:"enabled"`
}

type ScriptResponse struct {
	ID          string    `json:"id"`
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Runtime     string    `json:"runtime"`
	Actions     []string  `json:"actions"`
	SourceURL   string    `json:"source_url"`
	Enabled     bool      `json:"enabled"`
	Status      string    `json:"status"`
	InstalledAt time.Time `json:"installed_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}
