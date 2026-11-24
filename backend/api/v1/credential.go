package v1

import "time"

type CredentialCreateRequest struct {
	Platform  string            `json:"platform" binding:"required"`
	Data      map[string]string `json:"data" binding:"required"`
	Label     string            `json:"label"`
	IsDefault bool              `json:"is_default"`
	// Proxy 可选：http/socks 代理，仅用于该凭据的任务与发现。
	Proxy string `json:"proxy"`
}

type CredentialUpdateRequest struct {
	// Data 为空时仅更新元信息（名称/默认标记），不改动密文。
	Data      map[string]string `json:"data"`
	Label     string            `json:"label"`
	IsDefault bool              `json:"is_default"`
	// Proxy 非空时替换；ClearProxy 为 true 时清除。
	Proxy      string `json:"proxy"`
	ClearProxy bool   `json:"clear_proxy"`
}

type CredentialResponse struct {
	ID         string     `json:"id"`
	Platform   string     `json:"platform"`
	Label      string     `json:"label"`
	Preview    string     `json:"preview"`
	ProxyHint  string     `json:"proxy_hint,omitempty"`
	IsDefault  bool       `json:"is_default"`
	LastUsedAt *time.Time `json:"last_used_at,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
	UpdatedAt  time.Time  `json:"updated_at"`
}

// CredentialAccessLogResponse 是凭据解密审计记录的对外视图（不含明文）。
type CredentialAccessLogResponse struct {
	ID           string    `json:"id"`
	Platform     string    `json:"platform"`
	CredentialID string    `json:"credential_id,omitempty"`
	Purpose      string    `json:"purpose"`
	RefID        string    `json:"ref_id,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}
