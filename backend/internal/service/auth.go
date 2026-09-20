package service

import (
	"context"
	"errors"
	"time"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/jwt"
	"backend/pkg/log"
	"backend/pkg/sid"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const tokenTTL = 7 * 24 * time.Hour

type AuthService struct {
	userRepo *repository.UserRepository
	sid      *sid.Sid
	jwt      *jwt.JWT
	logger   *log.Logger
}

func NewAuthService(userRepo *repository.UserRepository, sid *sid.Sid, jwt *jwt.JWT, logger *log.Logger) *AuthService {
	return &AuthService{userRepo: userRepo, sid: sid, jwt: jwt, logger: logger}
}

func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest) (*v1.LoginResponse, error) {
	if _, err := s.userRepo.GetByUsername(ctx, req.Username); err == nil {
		return nil, v1.ErrUserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, err
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	user := &model.User{ID: id, Username: req.Username, PasswordHash: string(hash), Role: "user"}
	if total, err := s.userRepo.Count(ctx); err == nil && total == 0 {
		user.Role = "admin" // 首个注册用户为管理员
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, err
	}
	return s.issue(user)
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest) (*v1.LoginResponse, error) {
	user, err := s.userRepo.GetByUsername(ctx, req.Username)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrInvalidCredentials
		}
		return nil, err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		return nil, v1.ErrInvalidCredentials
	}
	return s.issue(user)
}

func (s *AuthService) issue(user *model.User) (*v1.LoginResponse, error) {
	expiresAt := time.Now().Add(tokenTTL)
	token, err := s.jwt.GenToken(user.ID, user.Role, expiresAt)
	if err != nil {
		return nil, err
	}
	return &v1.LoginResponse{
		Token:     token,
		ExpiresAt: expiresAt.Unix(),
		UserID:    user.ID,
		Username:  user.Username,
		Role:      user.Role,
	}, nil
}
