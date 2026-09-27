package service

import (
	"context"
	"encoding/json"
	"errors"

	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const (
	defaultMaxCredentials = 3
	defaultTokenHeadLen   = 6
	defaultTokenTailLen   = 4
)

// defaultPreviewFields 是用于展示/脱敏的标识字段（按优先级）。
var defaultPreviewFields = []string{
	"username", "account", "phone", "email", "user", "mobile", "student_id", "sid",
}

// PlatformConfigService 管理平台级配置；缺失时写入默认（启动 Bootstrap）。
type PlatformConfigService struct {
	repo       *repository.PlatformConfigRepository
	scriptRepo *repository.ScriptRepository
	logger     *log.Logger
}

func NewPlatformConfigService(repo *repository.PlatformConfigRepository, scriptRepo *repository.ScriptRepository, logger *log.Logger) *PlatformConfigService {
	return &PlatformConfigService{repo: repo, scriptRepo: scriptRepo, logger: logger}
}

func defaultConfig(platformID string) *model.PlatformConfig {
	fields, _ := json.Marshal(defaultPreviewFields)
	return &model.PlatformConfig{
		PlatformID:     platformID,
		MaxCredentials: defaultMaxCredentials,
		PreviewFields:  string(fields),
		TokenHeadLen:   defaultTokenHeadLen,
		TokenTailLen:   defaultTokenTailLen,
	}
}

// Ensure 返回平台配置，缺失则写入默认。
func (s *PlatformConfigService) Ensure(ctx context.Context, platformID string) *model.PlatformConfig {
	cfg, err := s.repo.Get(ctx, platformID)
	if err == nil {
		return cfg
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		s.logger.Warn("platform config get failed", zap.String("platform", platformID), zap.Error(err))
	}
	cfg = defaultConfig(platformID)
	if err := s.repo.Create(ctx, cfg); err != nil {
		s.logger.Warn("platform config create failed", zap.String("platform", platformID), zap.Error(err))
	}
	return cfg
}

// Bootstrap 为已安装脚本补齐默认配置（启动时调用）。
func (s *PlatformConfigService) Bootstrap(ctx context.Context) {
	scripts, err := s.scriptRepo.List(ctx)
	if err != nil {
		s.logger.Warn("platform config bootstrap failed", zap.Error(err))
		return
	}
	for i := range scripts {
		s.Ensure(ctx, scripts[i].ID)
	}
}
