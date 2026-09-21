package v1

import "time"

type PlatformField struct {
	Key     string `json:"key"`
	Label   string `json:"label"`
	Type    string `json:"type"`
	Default any    `json:"default,omitempty"`
}

type PlatformAction struct {
	ID   string `json:"id"`
	Name string `json:"name"`
}

type PlatformDiscovery struct {
	Label    string            `json:"label"`
	Columns  []PlatformField   `json:"columns"`
	Actions  []string          `json:"actions"`
	ParamMap map[string]string `json:"param_map"`
}

// PlatformResponse 是脚本 manifest 的完整对外 schema，供前端生成表单/筛选/展示。
type PlatformResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Runtime     string             `json:"runtime"`
	Status      string             `json:"status"`
	Actions     []PlatformAction   `json:"actions"`
	Params      []PlatformField    `json:"params"`
	Credentials []PlatformField    `json:"credentials"`
	Discovery   *PlatformDiscovery `json:"discovery"`
}

type DiscoveryItem struct {
	Key    string            `json:"key"`
	Fields map[string]string `json:"fields"`
}

type DiscoveryResponse struct {
	Platform    string          `json:"platform"`
	Label       string          `json:"label"`
	Columns     []PlatformField `json:"columns"`
	Actions     []string        `json:"actions"`
	Items       []DiscoveryItem `json:"items"`
	RefreshedAt *time.Time      `json:"refreshed_at"`
}
