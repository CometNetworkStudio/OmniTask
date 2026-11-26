package repository

import (
	"context"
	"errors"
	"time"

	"backend/internal/model"
	"gorm.io/gorm"
)

type SessionRepository struct{ r *Repository }

func NewSessionRepository(r *Repository) *SessionRepository { return &SessionRepository{r: r} }

func (repo *SessionRepository) Create(ctx context.Context, session *model.Session) error {
	return repo.r.DB(ctx).Create(session).Error
}

func (repo *SessionRepository) GetByTokenHash(ctx context.Context, hash string) (*model.Session, error) {
	var session model.Session
	if err := repo.r.DB(ctx).Where("token_hash = ?", hash).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

func (repo *SessionRepository) GetByID(ctx context.Context, id string) (*model.Session, error) {
	var session model.Session
	if err := repo.r.DB(ctx).Where("id = ?", id).First(&session).Error; err != nil {
		return nil, err
	}
	return &session, nil
}

// Touch 只更新最近活跃时间；调用方应做节流，避免每个请求都写库。
func (repo *SessionRepository) Touch(ctx context.Context, id string, at time.Time) error {
	return repo.r.DB(ctx).Model(&model.Session{}).Where("id = ?", id).
		Update("last_seen_at", at).Error
}

func (repo *SessionRepository) Revoke(ctx context.Context, id string, at time.Time) error {
	return repo.r.DB(ctx).Model(&model.Session{}).
		Where("id = ? AND revoked_at IS NULL", id).
		Update("revoked_at", at).Error
}

// RevokeByUser 吊销用户全部未吊销会话；exceptID 非空时保留该会话。
func (repo *SessionRepository) RevokeByUser(ctx context.Context, userID, exceptID string, at time.Time) error {
	q := repo.r.DB(ctx).Model(&model.Session{}).
		Where("user_id = ? AND revoked_at IS NULL", userID)
	if exceptID != "" {
		q = q.Where("id <> ?", exceptID)
	}
	return q.Update("revoked_at", at).Error
}

func (repo *SessionRepository) ListActiveByUser(ctx context.Context, userID string, now time.Time) ([]model.Session, error) {
	var sessions []model.Session
	if err := repo.r.DB(ctx).
		Where("user_id = ? AND revoked_at IS NULL AND expires_at > ?", userID, now).
		Order("last_seen_at DESC").Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (repo *SessionRepository) ListActive(ctx context.Context, now time.Time, limit int) ([]model.Session, error) {
	if limit <= 0 || limit > 500 {
		limit = 100
	}
	var sessions []model.Session
	if err := repo.r.DB(ctx).
		Where("revoked_at IS NULL AND expires_at > ?", now).
		Order("last_seen_at DESC").Limit(limit).Find(&sessions).Error; err != nil {
		return nil, err
	}
	return sessions, nil
}

func (repo *SessionRepository) CountActive(ctx context.Context, now time.Time) (int64, error) {
	var count int64
	err := repo.r.DB(ctx).Model(&model.Session{}).
		Where("revoked_at IS NULL AND expires_at > ?", now).Count(&count).Error
	return count, err
}

// DeleteExpired 只清理已过期会话；已吊销会话保留到过期，便于排查。
func (repo *SessionRepository) DeleteExpired(ctx context.Context, now time.Time) error {
	return repo.r.DB(ctx).Where("expires_at <= ?", now).Delete(&model.Session{}).Error
}

type LoginAttemptRepository struct{ r *Repository }

func NewLoginAttemptRepository(r *Repository) *LoginAttemptRepository {
	return &LoginAttemptRepository{r: r}
}

func (repo *LoginAttemptRepository) Create(ctx context.Context, attempt *model.LoginAttempt) error {
	return repo.r.DB(ctx).Create(attempt).Error
}

func (repo *LoginAttemptRepository) CountFailuresSince(ctx context.Context, username string, since time.Time) (int64, error) {
	var count int64
	err := repo.r.DB(ctx).Model(&model.LoginAttempt{}).
		Where("username = ? AND success = ? AND created_at >= ?", username, false, since).
		Count(&count).Error
	return count, err
}

func (repo *LoginAttemptRepository) CountFailuresTotalSince(ctx context.Context, since time.Time) (int64, error) {
	var count int64
	err := repo.r.DB(ctx).Model(&model.LoginAttempt{}).
		Where("success = ? AND created_at >= ?", false, since).
		Count(&count).Error
	return count, err
}

// LastSuccessAt 返回该用户名最近一次成功登录时间，用于登录成功后重置失败计数。
func (repo *LoginAttemptRepository) LastSuccessAt(ctx context.Context, username string) (*time.Time, error) {
	var attempt model.LoginAttempt
	err := repo.r.DB(ctx).
		Where("username = ? AND success = ?", username, true).
		Order("created_at DESC").First(&attempt).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, nil
		}
		return nil, err
	}
	return &attempt.CreatedAt, nil
}

func (repo *LoginAttemptRepository) ListRecent(ctx context.Context, limit int) ([]model.LoginAttempt, error) {
	if limit <= 0 || limit > 200 {
		limit = 50
	}
	var attempts []model.LoginAttempt
	if err := repo.r.DB(ctx).Order("created_at DESC").Limit(limit).Find(&attempts).Error; err != nil {
		return nil, err
	}
	return attempts, nil
}

func (repo *LoginAttemptRepository) DeleteOlderThan(ctx context.Context, cutoff time.Time) error {
	return repo.r.DB(ctx).Where("created_at < ?", cutoff).Delete(&model.LoginAttempt{}).Error
}
