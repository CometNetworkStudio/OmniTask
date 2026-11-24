package v1

import "time"

type PlatformOption struct {
	Label string `json:"label"`
	Value string `json:"value"`
}

type PlatformField struct {
	Key         string           `json:"key"`
	Label       string           `json:"label"`
	Type        string           `json:"type"`
	Default     any              `json:"default,omitempty"`
	Required    bool             `json:"required"`
	Placeholder string           `json:"placeholder,omitempty"`
	Help        string           `json:"help,omitempty"`
	Options     []PlatformOption `json:"options,omitempty"`
	Min         any              `json:"min,omitempty"`
	Max         any              `json:"max,omitempty"`
	Step        any              `json:"step,omitempty"`
	Multiline   bool             `json:"multiline"`
}

type PlatformResult struct {
	Type    string          `json:"type"`
	Columns []PlatformField `json:"columns"`
}

type PlatformAction struct {
	ID          string          `json:"id"`
	Name        string          `json:"name"`
	Description string          `json:"description"`
	Params      []PlatformField `json:"params"`
	Result      *PlatformResult `json:"result"`
}

type PlatformDiscovery struct {
	Label    string            `json:"label"`
	Columns  []PlatformField   `json:"columns"`
	Actions  []string          `json:"actions"`
	ParamMap map[string]string `json:"param_map"`
	Primary  string            `json:"primary"` // 卡片视图主字段（列 key），缺省用第一列
}

// PlatformResponse 是脚本 manifest 的完整对外 schema，供前端生成表单/筛选/展示。
type PlatformResponse struct {
	ID          string             `json:"id"`
	Name        string             `json:"name"`
	Version     string             `json:"version"`
	Icon        string             `json:"icon"`
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
