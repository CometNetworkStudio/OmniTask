package service

import (
	"context"
	"errors"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

type UserService struct {
	repo   *repository.UserRepository
	conf   *viper.Viper
	logger *log.Logger
}

func NewUserService(repo *repository.UserRepository, conf *viper.Viper, logger *log.Logger) *UserService {
	return &UserService{repo: repo, conf: conf, logger: logger}
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
	return s.Me(ctx, id)
}

func toUserResponse(user *model.User) *v1.UserResponse {
	return &v1.UserResponse{
		ID:        user.ID,
		Username:  user.Username,
		Role:      user.Role,
		Enabled:   user.Enabled,
		CreatedAt: user.CreatedAt,
	}
}
