package service

import (
	"context"
	"encoding/json"
	"errors"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type UserService struct {
	repo     *repository.UserRepository
	sessions *SessionService
	conf     *viper.Viper
	logger   *log.Logger
}

func NewUserService(repo *repository.UserRepository, sessions *SessionService, conf *viper.Viper, logger *log.Logger) *UserService {
	return &UserService{repo: repo, sessions: sessions, conf: conf, logger: logger}
}

func (s *UserService) Me(ctx context.Context, userID string) (*v1.UserResponse, error) {
	user, err := s.repo.GetByID(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrNotFound
		}
		return nil, err
	}
	return toUserResponse(user), nil
}

func (s *UserService) List(ctx context.Context, page, size int) ([]v1.UserResponse, int64, error) {
	page, size = normalizePage(s.conf, page, size)
	users, total, err := s.repo.List(ctx, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]v1.UserResponse, 0, len(users))
	for i := range users {
		out = append(out, *toUserResponse(&users[i]))
	}
	return out, total, nil
}

func (s *UserService) UpdateRole(ctx context.Context, id, role string) (*v1.UserResponse, error) {
	if role != "admin" && role != "user" {
		return nil, v1.ErrBadRequest
	}
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrNotFound
		}
		return nil, err
	}
	if err := s.repo.UpdateRole(ctx, id, role); err != nil {
		return nil, err
	}
	return s.Me(ctx, id)
}

func (s *UserService) SetEnabled(ctx context.Context, id string, enabled bool) (*v1.UserResponse, error) {
	if _, err := s.repo.GetByID(ctx, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrNotFound
		}
		return nil, err
	}
	if err := s.repo.SetEnabled(ctx, id, enabled); err != nil {
		return nil, err
	}
	// 禁用账号立即失效所有会话，不再依赖令牌过期。
	if !enabled && s.sessions != nil {
		if err := s.sessions.RevokeUser(ctx, id); err != nil {
			s.logger.Warn("revoke disabled user sessions failed: " + err.Error())
		}
	}
	return s.Me(ctx, id)
}

// Preferences 返回用户界面偏好；没有记录时返回空列表。
func (s *UserService) Preferences(ctx context.Context, userID string) (*v1.UserPreferences, error) {
	pref, err := s.repo.GetPreference(ctx, userID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &v1.UserPreferences{PinnedPlatforms: []string{}}, nil
		}
		return nil, err
	}
	ids := []string{}
	if pref.PinnedPlatforms != "" {
		_ = json.Unmarshal([]byte(pref.PinnedPlatforms), &ids)
	}
	if ids == nil {
		ids = []string{}
	}
	return &v1.UserPreferences{PinnedPlatforms: ids}, nil
}

// UpdatePreferences 覆盖保存用户置顶平台列表。
func (s *UserService) UpdatePreferences(ctx context.Context, userID string, req *v1.UpdatePreferencesRequest) (*v1.UserPreferences, error) {
	ids := sanitizePinnedPlatforms(req.PinnedPlatforms)
	raw, err := json.Marshal(ids)
	if err != nil {
		return nil, err
	}
	pref := &model.UserPreference{
		UserID:          userID,
		PinnedPlatforms: string(raw),
		UpdatedAt:       time.Now(),
	}
	if err := s.repo.SavePreference(ctx, pref); err != nil {
		return nil, err
	}
	return &v1.UserPreferences{PinnedPlatforms: ids}, nil
}

func sanitizePinnedPlatforms(values []string) []string {
	seen := make(map[string]bool, len(values))
	out := make([]string, 0, len(values))
	for _, value := range values {
		value = strings.TrimSpace(value)
		if value == "" || len(value) > 64 || seen[value] {
			continue
		}
		seen[value] = true
		out = append(out, value)
		if len(out) >= 100 {
			break
		}
	}
	return out
}

func toUserResponse(user *model.User) *v1.UserResponse {
	return &v1.UserResponse{
		ID:              user.ID,
		Username:        user.Username,
		Role:            user.Role,
		Enabled:         user.Enabled,
		TermsVersion:    user.TermsVersion,
		TermsAcceptedAt: user.TermsAcceptedAt,
		CreatedAt:       user.CreatedAt,
	}
}
