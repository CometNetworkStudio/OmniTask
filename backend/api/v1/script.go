package v1

import "time"

type ScriptInstallRequest struct {
	URL string `json:"url" binding:"required"`
	// Ref 可选：分支/标签/commit。为空则默认分支 main。
	Ref string `json:"ref"`
}

type ScriptToggleRequest struct {
	Enabled bool `json:"enabled"`
}

type ScriptDebugRequest struct {
	Action      string            `json:"action" binding:"required"`
	Params      map[string]string `json:"params"`
	Credentials map[string]string `json:"credentials"`
	Proxy       string            `json:"proxy"`
}

type ScriptDebugEvent struct {
	Kind       string `json:"kind"`
	Percent    int    `json:"percent"`
	Message    string `json:"message"`
	Level      string `json:"level"`
	Step       string `json:"step"`
	Label      string `json:"label"`
	StepStatus string `json:"step_status"`
	Current    int    `json:"current"`
	Total      int    `json:"total"`
	Unit       string `json:"unit"`
	Key        string `json:"key"`
	Value      string `json:"value"`
	Stage      string `json:"stage"`
	Timestamp  int64  `json:"timestamp"`
	Result     any    `json:"result,omitempty"`
}

type ScriptResponse struct {
	ID           string    `json:"id"`
	Name         string    `json:"name"`
	Version      string    `json:"version"`
	Author       string    `json:"author"`
	Icon         string    `json:"icon"`
	Runtime      string    `json:"runtime"`
	Actions      []string  `json:"actions"`
	SourceURL    string    `json:"source_url"`
	Ref          string    `json:"ref"`
	CommitSHA    string    `json:"commit_sha"`
	ManifestHash string    `json:"manifest_hash"`
	Manifest     string    `json:"manifest"`
	LastError    string    `json:"last_error"`
	HasBackup    bool      `json:"has_backup"`
	Enabled      bool      `json:"enabled"`
	Status       string    `json:"status"`
	InstalledAt  time.Time `json:"installed_at"`
	UpdatedAt    time.Time `json:"updated_at"`
}
