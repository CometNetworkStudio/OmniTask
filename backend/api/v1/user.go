package v1

import "time"

type UserResponse struct {
	ID              string     `json:"id"`
	Username        string     `json:"username"`
	Role            string     `json:"role"`
	Enabled         bool       `json:"enabled"`
	TermsVersion    string     `json:"terms_version"`
	TermsAcceptedAt *time.Time `json:"terms_accepted_at"`
	CreatedAt       time.Time  `json:"created_at"`
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
