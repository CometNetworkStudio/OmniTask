package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/authcookie"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

const sessionTouchInterval = 5 * time.Minute

// SessionService 负责登录会话与登录尝试。Cookie 中只放随机令牌，库里存摘要。
type SessionService struct {
	sessions *repository.SessionRepository
	attempts *repository.LoginAttemptRepository
	users    *repository.UserRepository
	sid      *sid.Sid
	conf     *viper.Viper
	logger   *log.Logger
}

func NewSessionService(
	sessions *repository.SessionRepository,
	attempts *repository.LoginAttemptRepository,
	users *repository.UserRepository,
	sid *sid.Sid,
	conf *viper.Viper,
	logger *log.Logger,
) *SessionService {
	return &SessionService{
		sessions: sessions,
		attempts: attempts,
		users:    users,
		sid:      sid,
		conf:     conf,
		logger:   logger,
	}
}

func hashToken(token string) string {
	sum := sha256.Sum256([]byte(token))
	return hex.EncodeToString(sum[:])
}

func (s *SessionService) ttl() time.Duration {
	hours := s.conf.GetInt("security.session_ttl_hours")
	if hours <= 0 {
		hours = 168
	}
	return time.Duration(hours) * time.Hour
}

// Create 新建会话，返回仅此一次可见的明文令牌。
func (s *SessionService) Create(ctx context.Context, userID, ip, userAgent string) (string, *model.Session, error) {
	token, err := authcookie.NewToken()
	if err != nil {
		return "", nil, err
	}
	id, err := s.sid.GenString()
	if err != nil {
		return "", nil, err
	}
	now := time.Now()
	session := &model.Session{
		ID:         id,
		UserID:     userID,
		TokenHash:  hashToken(token),
		IP:         truncate(ip, 64),
		UserAgent:  truncate(userAgent, 255),
		CreatedAt:  now,
		LastSeenAt: now,
		ExpiresAt:  now.Add(s.ttl()),
	}
	if err := s.sessions.Create(ctx, session); err != nil {
		return "", nil, err
	}
	return token, session, nil
}

// Validate 校验令牌并返回当前用户；禁用账号、已吊销、已过期都会立即失败。
func (s *SessionService) Validate(ctx context.Context, token string) (*model.User, *model.Session, error) {
	if strings.TrimSpace(token) == "" {
		return nil, nil, v1.ErrUnauthorized
	}
	session, err := s.sessions.GetByTokenHash(ctx, hashToken(token))
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, v1.ErrUnauthorized
		}
		return nil, nil, err
	}
	now := time.Now()
	if session.RevokedAt != nil || !now.Before(session.ExpiresAt) {
		return nil, nil, v1.ErrUnauthorized
	}
	user, err := s.users.GetByID(ctx, session.UserID)
	if err != nil {
		_ = s.sessions.Revoke(ctx, session.ID, now)
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil, v1.ErrUnauthorized
		}
		return nil, nil, err
	}
	if !user.Enabled {
		_ = s.sessions.Revoke(ctx, session.ID, now)
		return nil, nil, v1.ErrUnauthorized
	}
	if now.Sub(session.LastSeenAt) >= sessionTouchInterval {
		if err := s.sessions.Touch(ctx, session.ID, now); err == nil {
			session.LastSeenAt = now
		}
	}
	return user, session, nil
}

func (s *SessionService) Revoke(ctx context.Context, sessionID string) error {
	return s.sessions.Revoke(ctx, sessionID, time.Now())
}

// RevokeOwned 只允许吊销自己的会话，避免越权操作。
func (s *SessionService) RevokeOwned(ctx context.Context, userID, sessionID string) error {
	session, err := s.sessions.GetByID(ctx, sessionID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return v1.ErrSessionNotFound
		}
		return err
	}
	if session.UserID != userID {
		return v1.ErrSessionNotFound
	}
	return s.sessions.Revoke(ctx, session.ID, time.Now())
}

// RevokeByID 是管理员入口，可吊销任意会话。
func (s *SessionService) RevokeByID(ctx context.Context, sessionID string) error {
	if _, err := s.sessions.GetByID(ctx, sessionID); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return v1.ErrSessionNotFound
		}
		return err
	}
	return s.sessions.Revoke(ctx, sessionID, time.Now())
}

func (s *SessionService) RevokeUser(ctx context.Context, userID string) error {
	return s.sessions.RevokeByUser(ctx, userID, "", time.Now())
}

// RevokeOthers 保留当前会话，退出其余设备。
func (s *SessionService) RevokeOthers(ctx context.Context, userID, currentID string) error {
	return s.sessions.RevokeByUser(ctx, userID, currentID, time.Now())
}

func (s *SessionService) ListUser(ctx context.Context, userID, currentID string) ([]v1.SessionResponse, error) {
	sessions, err := s.sessions.ListActiveByUser(ctx, userID, time.Now())
	if err != nil {
		return nil, err
	}
	username := ""
	if user, err := s.users.GetByID(ctx, userID); err == nil {
		username = user.Username
	}
	out := make([]v1.SessionResponse, 0, len(sessions))
	for i := range sessions {
		out = append(out, toSessionResponse(&sessions[i], username, currentID))
	}
	return out, nil
}

func (s *SessionService) ListAll(ctx context.Context, currentID string, limit int) ([]v1.SessionResponse, error) {
	sessions, err := s.sessions.ListActive(ctx, time.Now(), limit)
	if err != nil {
		return nil, err
	}
	ids := make([]string, 0, len(sessions))
	seen := map[string]bool{}
	for i := range sessions {
		if !seen[sessions[i].UserID] {
			seen[sessions[i].UserID] = true
			ids = append(ids, sessions[i].UserID)
		}
	}
	users, err := s.users.GetByIDs(ctx, ids)
	if err != nil {
		return nil, err
	}
	names := make(map[string]string, len(users))
	for i := range users {
		names[users[i].ID] = users[i].Username
	}
	out := make([]v1.SessionResponse, 0, len(sessions))
	for i := range sessions {
		out = append(out, toSessionResponse(&sessions[i], names[sessions[i].UserID], currentID))
	}
	return out, nil
}

func toSessionResponse(session *model.Session, username, currentID string) v1.SessionResponse {
	return v1.SessionResponse{
		ID:         session.ID,
		UserID:     session.UserID,
		Username:   username,
		IP:         session.IP,
		UserAgent:  session.UserAgent,
		CreatedAt:  session.CreatedAt,
		LastSeenAt: session.LastSeenAt,
		ExpiresAt:  session.ExpiresAt,
		Current:    session.ID == currentID,
	}
}

// Locked 判断账号在窗口内的失败次数是否已达上限。数据库计数，多实例也一致。
func (s *SessionService) Locked(ctx context.Context, username string) bool {
	max := s.conf.GetInt("security.login_max_failures")
	if max <= 0 {
		return false
	}
	window := s.conf.GetInt("security.login_window_minutes")
	if window <= 0 {
		window = 15
	}
	username = strings.TrimSpace(username)
	since := time.Now().Add(-time.Duration(window) * time.Minute)
	if last, err := s.attempts.LastSuccessAt(ctx, username); err == nil && last != nil && last.After(since) {
		since = *last
	}
	count, err := s.attempts.CountFailuresSince(ctx, username, since)
	if err != nil {
		s.logger.Warn("count login failures failed: " + err.Error())
		return false
	}
	return count >= int64(max)
}

func (s *SessionService) RecordAttempt(ctx context.Context, username, ip, userAgent string, success bool, reason string) {
	id, err := s.sid.GenString()
	if err != nil {
		s.logger.Warn("generate login attempt id failed: " + err.Error())
		return
	}
	attempt := &model.LoginAttempt{
		ID:        id,
		Username:  truncate(strings.TrimSpace(username), 64),
		IP:        truncate(ip, 64),
		UserAgent: truncate(userAgent, 255),
		Success:   success,
		Reason:    truncate(reason, 64),
		CreatedAt: time.Now(),
	}
	if err := s.attempts.Create(ctx, attempt); err != nil {
		s.logger.Warn("record login attempt failed: " + err.Error())
	}
}

func (s *SessionService) ListAttempts(ctx context.Context, limit int) ([]v1.LoginAttemptResponse, error) {
	attempts, err := s.attempts.ListRecent(ctx, limit)
	if err != nil {
		return nil, err
	}
	out := make([]v1.LoginAttemptResponse, 0, len(attempts))
	for i := range attempts {
		out = append(out, v1.LoginAttemptResponse{
			ID:        attempts[i].ID,
			Username:  attempts[i].Username,
			IP:        attempts[i].IP,
			UserAgent: attempts[i].UserAgent,
			Success:   attempts[i].Success,
			Reason:    attempts[i].Reason,
			CreatedAt: attempts[i].CreatedAt,
		})
	}
	return out, nil
}

func (s *SessionService) Overview(ctx context.Context) (*v1.SecurityOverview, error) {
	registration := strings.TrimSpace(s.conf.GetString("security.registration"))
	if registration == "" {
		registration = "open"
	}
	ttlHours := s.conf.GetInt("security.session_ttl_hours")
	if ttlHours <= 0 {
		ttlHours = 168
	}
	window := s.conf.GetInt("security.login_window_minutes")
	if window <= 0 {
		window = 15
	}
	userCount, err := s.users.Count(ctx)
	if err != nil {
		return nil, err
	}
	active, err := s.sessions.CountActive(ctx, time.Now())
	if err != nil {
		return nil, err
	}
	failed, err := s.attempts.CountFailuresTotalSince(ctx, time.Now().Add(-24*time.Hour))
	if err != nil {
		return nil, err
	}
	return &v1.SecurityOverview{
		Registration:       registration,
		InviteRequired:     registration == "invite",
		SessionTTLHours:    ttlHours,
		LoginMaxFailures:   s.conf.GetInt("security.login_max_failures"),
		LoginWindowMinutes: window,
		MaxActiveTasks:     s.conf.GetInt("task.max_active_per_user"),
		DailyTasks:         s.conf.GetInt("task.daily_tasks_per_user"),
		MaxPendingTasks:    s.conf.GetInt("task.max_pending_per_user"),
		UserCount:          userCount,
		ActiveSessions:     active,
		FailedLogins24h:    failed,
	}, nil
}

// Cleanup 删除过期会话和过期登录记录；启动与登录时调用，避免后台常驻任务。
func (s *SessionService) Cleanup(ctx context.Context) {
	now := time.Now()
	if err := s.sessions.DeleteExpired(ctx, now); err != nil {
		s.logger.Warn("cleanup expired sessions failed: " + err.Error())
	}
	if err := s.attempts.DeleteOlderThan(ctx, now.Add(-7*24*time.Hour)); err != nil {
		s.logger.Warn("cleanup login attempts failed: " + err.Error())
	}
}

func truncate(value string, max int) string {
	runes := []rune(value)
	if len(runes) <= max {
		return value
	}
	return string(runes[:max])
}
