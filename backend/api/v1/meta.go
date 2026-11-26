package v1

// MetaLimits 只暴露对前端有用的配额，不含任何密钥或内部配置。
type MetaLimits struct {
	MaxActiveTasks  int `json:"max_active_tasks"`
	DailyTasks      int `json:"daily_tasks"`
	MaxPendingTasks int `json:"max_pending_tasks"`
	SessionTTLHours int `json:"session_ttl_hours"`
}

// MetaResponse 让前端按部署能力渲染，自托管和官方托管可以共用同一套代码。
type MetaResponse struct {
	Version           string          `json:"version"`
	Env               string          `json:"env"`
	Registration      string          `json:"registration"`
	InviteRequired    bool            `json:"invite_required"`
	Captcha           bool            `json:"captcha"`
	EmailVerification bool            `json:"email_verification"`
	Features          map[string]bool `json:"features"`
	Limits            MetaLimits      `json:"limits"`
}
