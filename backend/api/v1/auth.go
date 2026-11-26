package v1

type RegisterRequest struct {
	Username   string `json:"username" binding:"required,min=3,max=64"`
	Password   string `json:"password" binding:"required,min=8,max=128"`
	InviteCode string `json:"invite_code" binding:"max=128"`
}

type LoginRequest struct {
	Username string `json:"username" binding:"required"`
	Password string `json:"password" binding:"required"`
}

// LoginResponse 不再返回 JWT，认证状态由 HttpOnly 会话 Cookie 承载。
type LoginResponse struct {
	User      UserResponse `json:"user"`
	ExpiresAt int64        `json:"expires_at"`
}
