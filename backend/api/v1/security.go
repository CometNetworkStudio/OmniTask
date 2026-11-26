package v1

import "time"

// SessionResponse 是会话列表的展示数据，绝不包含令牌或其摘要。
type SessionResponse struct {
	ID         string    `json:"id"`
	UserID     string    `json:"user_id"`
	Username   string    `json:"username"`
	IP         string    `json:"ip"`
	UserAgent  string    `json:"user_agent"`
	CreatedAt  time.Time `json:"created_at"`
	LastSeenAt time.Time `json:"last_seen_at"`
	ExpiresAt  time.Time `json:"expires_at"`
	Current    bool      `json:"current"`
}

type LoginAttemptResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Success   bool      `json:"success"`
	Reason    string    `json:"reason"`
	CreatedAt time.Time `json:"created_at"`
}

// SecurityOverview 供管理员安全页展示当前部署的安全参数与计数。
type SecurityOverview struct {
	Registration       string `json:"registration"`
	InviteRequired     bool   `json:"invite_required"`
	SessionTTLHours    int    `json:"session_ttl_hours"`
	LoginMaxFailures   int    `json:"login_max_failures"`
	LoginWindowMinutes int    `json:"login_window_minutes"`
	MaxActiveTasks     int    `json:"max_active_tasks"`
	DailyTasks         int    `json:"daily_tasks"`
	MaxPendingTasks    int    `json:"max_pending_tasks"`
	UserCount          int64  `json:"user_count"`
	ActiveSessions     int64  `json:"active_sessions"`
	FailedLogins24h    int64  `json:"failed_logins_24h"`
}
