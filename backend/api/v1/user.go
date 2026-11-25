package v1

import "time"

type UserResponse struct {
	ID        string    `json:"id"`
	Username  string    `json:"username"`
	Role      string    `json:"role"`
	Enabled   bool      `json:"enabled"`
	CreatedAt time.Time `json:"created_at"`
}

type UpdateRoleRequest struct {
	Role string `json:"role" binding:"required,oneof=admin user"`
}

type SetEnabledRequest struct {
	Enabled bool `json:"enabled"`
}

type UserPreferences struct {
	PinnedPlatforms []string `json:"pinned_platforms"`
}

type UpdatePreferencesRequest struct {
	PinnedPlatforms []string `json:"pinned_platforms"`
}
