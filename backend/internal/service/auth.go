package service

import (
	"context"
	"crypto/subtle"
	"errors"
	"os"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthService struct {
	userRepo *repository.UserRepository
	sessions *SessionService
	sid      *sid.Sid
	conf     *viper.Viper
	logger   *log.Logger
	dummy    []byte
}

func NewAuthService(
	userRepo *repository.UserRepository,
	sessions *SessionService,
	sid *sid.Sid,
	conf *viper.Viper,
	logger *log.Logger,
) *AuthService {
	// 用于不存在用户时也执行一次 bcrypt，避免通过响应时间枚举用户名。
	dummy, _ := bcrypt.GenerateFromPassword([]byte("omnitask-dummy-password"), bcrypt.DefaultCost)
	return &AuthService{
		userRepo: userRepo,
		sessions: sessions,
		sid:      sid,
		conf:     conf,
		logger:   logger,
		dummy:    dummy,
	}
}

// Register 创建用户并直接建立会话。invite 模式下必须提供正确邀请码。
func (s *AuthService) Register(ctx context.Context, req *v1.RegisterRequest, ip, userAgent string) (*v1.LoginResponse, string, error) {
	if !req.AcceptTerms || strings.TrimSpace(req.TermsVersion) != v1.CurrentTermsVersion {
		return nil, "", v1.ErrTermsRequired
	}
	mode := strings.TrimSpace(s.conf.GetString("security.registration"))
	if mode == "" {
		mode = "open"
	}
	switch mode {
	case "closed":
		return nil, "", v1.ErrRegistrationClosed
	case "invite":
		code := strings.TrimSpace(s.conf.GetString("security.invite_code"))
		provided := strings.TrimSpace(req.InviteCode)
		if code == "" || subtle.ConstantTimeCompare([]byte(provided), []byte(code)) != 1 {
			return nil, "", v1.ErrInviteInvalid
		}
	case "open":
	default:
		return nil, "", v1.ErrRegistrationClosed
	}

	username := strings.TrimSpace(req.Username)
	if len(username) < 3 || len(username) > 64 {
		return nil, "", v1.ErrBadRequest
	}
	if len(req.Password) < 8 || len(req.Password) > 128 {
		return nil, "", v1.ErrBadRequest
	}
	if _, err := s.userRepo.GetByUsername(ctx, username); err == nil {
		return nil, "", v1.ErrUserAlreadyExists
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(req.Password), bcrypt.DefaultCost)
	if err != nil {
		return nil, "", err
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, "", err
	}
	now := time.Now()
	user := &model.User{
		ID: id, Username: username, PasswordHash: string(hash),
		Role: "user", Enabled: true,
		TermsVersion: v1.CurrentTermsVersion, TermsAcceptedAt: &now,
	}
	if total, err := s.userRepo.Count(ctx); err == nil && total == 0 {
		if s.conf.GetBool("security.allow_first_user_admin") {
			user.Role = "admin" // 自托管默认保留首个用户为管理员
		}
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		return nil, "", err
	}
	return s.issue(ctx, user, ip, userAgent)
}

func (s *AuthService) Login(ctx context.Context, req *v1.LoginRequest, ip, userAgent string) (*v1.LoginResponse, string, error) {
	username := strings.TrimSpace(req.Username)
	if s.sessions.Locked(ctx, username) {
		s.sessions.RecordAttempt(ctx, username, ip, userAgent, false, "locked")
		return nil, "", v1.ErrTooManyRequests
	}
	user, err := s.userRepo.GetByUsername(ctx, username)
	if err != nil {
		_ = bcrypt.CompareHashAndPassword(s.dummy, []byte(req.Password))
		if errors.Is(err, gorm.ErrRecordNotFound) {
			s.sessions.RecordAttempt(ctx, username, ip, userAgent, false, "unknown_user")
			return nil, "", v1.ErrInvalidCredentials
		}
		return nil, "", err
	}
	if bcrypt.CompareHashAndPassword([]byte(user.PasswordHash), []byte(req.Password)) != nil {
		s.sessions.RecordAttempt(ctx, username, ip, userAgent, false, "bad_password")
		return nil, "", v1.ErrInvalidCredentials
	}
	if !user.Enabled {
		s.sessions.RecordAttempt(ctx, username, ip, userAgent, false, "disabled")
		return nil, "", v1.ErrAccountDisabled
	}
	resp, token, err := s.issue(ctx, user, ip, userAgent)
	if err != nil {
		return nil, "", err
	}
	s.sessions.RecordAttempt(ctx, username, ip, userAgent, true, "success")
	return resp, token, nil
}

func (s *AuthService) issue(ctx context.Context, user *model.User, ip, userAgent string) (*v1.LoginResponse, string, error) {
	token, session, err := s.sessions.Create(ctx, user.ID, ip, userAgent)
	if err != nil {
		return nil, "", err
	}
	return &v1.LoginResponse{
		User:      *toUserResponse(user),
		ExpiresAt: session.ExpiresAt.Unix(),
	}, token, nil
}

func (s *AuthService) Logout(ctx context.Context, sessionID string) error {
	if sessionID == "" {
		return nil
	}
	return s.sessions.Revoke(ctx, sessionID)
}

// BootstrapAdmin 在启动时按环境变量创建管理员，避免公开部署时依赖“第一个注册者”。
func (s *AuthService) BootstrapAdmin(ctx context.Context) {
	if strings.EqualFold(strings.TrimSpace(s.conf.GetString("security.registration")), "invite") &&
		strings.TrimSpace(s.conf.GetString("security.invite_code")) == "" {
		s.logger.Warn("注册模式为 invite 但 security.invite_code 为空，所有注册都会被拒绝")
	}
	username := strings.TrimSpace(os.Getenv("OMNITASK_ADMIN_USER"))
	password := os.Getenv("OMNITASK_ADMIN_PASSWORD")
	if username == "" || password == "" {
		if !s.conf.GetBool("security.allow_first_user_admin") {
			if total, err := s.userRepo.Count(ctx); err == nil && total == 0 {
				s.logger.Warn("没有管理员且已关闭首个用户自动成为管理员；请设置 OMNITASK_ADMIN_USER/OMNITASK_ADMIN_PASSWORD")
			}
		}
		return
	}
	if len(password) < 8 {
		s.logger.Warn("OMNITASK_ADMIN_PASSWORD 少于 8 位，已忽略管理员初始化")
		return
	}
	existing, err := s.userRepo.GetByUsername(ctx, username)
	if err == nil {
		if existing.Role != "admin" {
			s.logger.Warn("OMNITASK_ADMIN_USER 已存在但不是管理员，未自动提权: " + username)
		}
		return
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Warn("读取管理员账号失败: " + err.Error())
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		s.logger.Warn("生成管理员密码哈希失败: " + err.Error())
		return
	}
	id, err := s.sid.GenString()
	if err != nil {
		s.logger.Warn("生成管理员 ID 失败: " + err.Error())
		return
	}
	user := &model.User{
		ID:           id,
		Username:     username,
		PasswordHash: string(hash),
		Role:         "admin",
		Enabled:      true,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
	if err := s.userRepo.Create(ctx, user); err != nil {
		s.logger.Warn("创建管理员失败: " + err.Error())
		return
	}
	s.logger.Info("已通过环境变量创建管理员: " + username)
}
