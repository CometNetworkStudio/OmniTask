package service

import (
	"context"
	"encoding/json"
	"errors"
	"net/url"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/crypto"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
	"gorm.io/gorm"
)

const proxyKey = "__proxy"

func cloneStringMap(in map[string]string) map[string]string {
	out := make(map[string]string, len(in)+1)
	for key, value := range in {
		out[key] = value
	}
	return out
}

func maskProxy(raw string) string {
	raw = strings.TrimSpace(raw)
	if raw == "" {
		return ""
	}
	u, err := url.Parse(raw)
	if err != nil || u.Host == "" || u.Scheme == "" {
		return "已配置"
	}
	if u.User != nil {
		return u.Scheme + "://***@" + u.Host
	}
	return u.Scheme + "://" + u.Host
}

// CredentialAccess 描述一次凭据解密的用途，供审计记录。
type CredentialAccess struct {
	Purpose string // task | discover | update
	RefID   string // task_id，discover/update 为空
}

type CredentialService struct {
	repo            *repository.CredentialRepository
	audit           *repository.CredentialAccessLogRepository
	taskRepo        *repository.TaskRepository
	cacheRepo       *repository.PlatformCacheRepository
	cfg             *PlatformConfigService
	keys            *crypto.Keyring
	sid             *sid.Sid
	logger          *log.Logger
	auditFailClosed bool
}

func NewCredentialService(
	repo *repository.CredentialRepository,
	audit *repository.CredentialAccessLogRepository,
	taskRepo *repository.TaskRepository,
	cacheRepo *repository.PlatformCacheRepository,
	cfg *PlatformConfigService,
	keys *crypto.Keyring,
	sid *sid.Sid,
	logger *log.Logger,
	conf *viper.Viper,
) *CredentialService {
	return &CredentialService{
		repo:            repo,
		audit:           audit,
		taskRepo:        taskRepo,
		cacheRepo:       cacheRepo,
		cfg:             cfg,
		keys:            keys,
		sid:             sid,
		logger:          logger,
		auditFailClosed: conf.GetBool("security.credential_audit_fail_closed"),
	}
}

func (s *CredentialService) List(ctx context.Context, userID string) ([]v1.CredentialResponse, error) {
	creds, err := s.repo.ListByUser(ctx, userID)
	if err != nil {
		return nil, err
	}
	out := make([]v1.CredentialResponse, 0, len(creds))
	for i := range creds {
		out = append(out, *toCredentialResponse(&creds[i]))
	}
	return out, nil
}

// Create 新增一份凭据（受平台 max_credentials 上限约束）。
func (s *CredentialService) Create(ctx context.Context, userID string, req *v1.CredentialCreateRequest) (*v1.CredentialResponse, error) {
	cfg := s.cfg.Ensure(ctx, req.Platform)
	count, err := s.repo.CountActive(ctx, userID, req.Platform)
	if err != nil {
		return nil, err
	}
	if int(count) >= cfg.MaxCredentials {
		return nil, v1.ErrCredentialLimit
	}
	data := cloneStringMap(req.Data)
	proxy := strings.TrimSpace(req.Proxy)
	if proxy != "" {
		data[proxyKey] = proxy
	} else {
		delete(data, proxyKey)
	}
	ciphertext, nonce, keyVersion, err := s.encrypt(data)
	if err != nil {
		return nil, err
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	isDefault := req.IsDefault || count == 0
	if isDefault {
		_ = s.repo.ClearDefault(ctx, userID, req.Platform)
	}
	cred := &model.Credential{
		ID: id, UserID: userID, Platform: req.Platform,
		Label: req.Label, Preview: s.buildPreview(cfg, data),
		ProxyHint: maskProxy(proxy), IsDefault: isDefault,
		KeyVersion: keyVersion, Cipher: ciphertext, Nonce: nonce,
	}
	if err := s.repo.Create(ctx, cred); err != nil {
		return nil, err
	}
	return toCredentialResponse(cred), nil
}

// Update 更新一份凭据的数据/名称/默认标记。
func (s *CredentialService) Update(ctx context.Context, userID, id string, req *v1.CredentialUpdateRequest) (*v1.CredentialResponse, error) {
	cred, err := s.repo.GetByID(ctx, userID, id)
	if err != nil || cred.ArchivedAt != nil {
		return nil, v1.ErrCredentialNotFound
	}
	cfg := s.cfg.Ensure(ctx, cred.Platform)
	proxyChanged := req.Proxy != "" || req.ClearProxy
	if len(req.Data) > 0 || proxyChanged {
		plain, err := s.keys.Decrypt(cred.Cipher, cred.Nonce, cred.KeyVersion)
		if err != nil {
			return nil, v1.ErrCredentialDecrypt
		}
		if err := s.auditAccess(ctx, userID, cred.Platform, cred.ID, CredentialAccess{Purpose: "update"}); err != nil && s.auditFailClosed {
			return nil, v1.ErrCredentialAudit
		}
		data := map[string]string{}
		if err := json.Unmarshal(plain, &data); err != nil {
			return nil, v1.ErrCredentialDecrypt
		}
		for key, value := range req.Data {
			data[key] = value
		}
		if req.ClearProxy {
			delete(data, proxyKey)
		} else if req.Proxy != "" {
			data[proxyKey] = strings.TrimSpace(req.Proxy)
		}
		ciphertext, nonce, keyVersion, err := s.encrypt(data)
		if err != nil {
			return nil, err
		}
		cred.Cipher = ciphertext
		cred.Nonce = nonce
		cred.KeyVersion = keyVersion
		cred.Preview = s.buildPreview(cfg, data)
		cred.ProxyHint = maskProxy(data[proxyKey])
	}
	if req.IsDefault {
		_ = s.repo.ClearDefault(ctx, userID, cred.Platform)
	}
	cred.Label = req.Label
	cred.IsDefault = req.IsDefault
	if err := s.repo.Update(ctx, cred); err != nil {
		return nil, err
	}
	return toCredentialResponse(cred), nil
}

// Archive 归档（或无可关联任务时真删）一份凭据，并清除其发现缓存。
func (s *CredentialService) Archive(ctx context.Context, userID, id string) error {
	cred, err := s.repo.GetByID(ctx, userID, id)
	if err != nil {
		return v1.ErrCredentialNotFound
	}
	if cred.ArchivedAt != nil {
		return nil
	}
	tasks, err := s.taskRepo.CountByCredential(ctx, id)
	if err != nil {
		return err
	}
	_ = s.cacheRepo.DeleteByCredential(ctx, id)
	if tasks == 0 {
		return s.repo.Delete(ctx, id)
	}
	return s.repo.SetArchived(ctx, id, time.Now())
}

// PurgeArchived 清理某平台下已无任何任务的归档凭据（在任务删除后调用）。
func (s *CredentialService) PurgeArchived(ctx context.Context, userID, platform string) {
	archived, err := s.repo.ListArchivedByPlatform(ctx, userID, platform)
	if err != nil {
		s.logger.Warn("list archived credentials failed", zap.Error(err))
		return
	}
	for i := range archived {
		n, err := s.taskRepo.CountByCredential(ctx, archived[i].ID)
		if err != nil || n > 0 {
			continue
		}
		_ = s.cacheRepo.DeleteByCredential(ctx, archived[i].ID)
		_ = s.repo.Delete(ctx, archived[i].ID)
	}
}

// GetOwned 校验凭据归属且未归档（供创建任务时使用）。
func (s *CredentialService) GetOwned(ctx context.Context, userID, id string) (*model.Credential, error) {
	cred, err := s.repo.GetByID(ctx, userID, id)
	if err != nil || cred.ArchivedAt != nil {
		return nil, v1.ErrCredentialNotFound
	}
	return cred, nil
}

// ResolveDefault 解析某平台的默认凭据（默认标记优先，其次唯一一条）。
func (s *CredentialService) ResolveDefault(ctx context.Context, userID, platform string) (*model.Credential, error) {
	cred, err := s.repo.GetDefault(ctx, userID, platform)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrCredentialNotFound
		}
		return nil, err
	}
	return cred, nil
}

// ListAccessLogs 返回某用户最近的凭据解密记录（仅元数据，用于透明度）。
func (s *CredentialService) ListAccessLogs(ctx context.Context, userID string, limit int) ([]v1.CredentialAccessLogResponse, error) {
	logs, err := s.audit.ListByUser(ctx, userID, limit)
	if err != nil {
		return nil, err
	}
	out := make([]v1.CredentialAccessLogResponse, 0, len(logs))
	for i := range logs {
		out = append(out, v1.CredentialAccessLogResponse{
			ID:           logs[i].ID,
			Platform:     logs[i].Platform,
			CredentialID: logs[i].CredentialID,
			Purpose:      logs[i].Purpose,
			RefID:        logs[i].RefID,
			CreatedAt:    logs[i].CreatedAt,
		})
	}
	return out, nil
}

// Decrypted 供任务下发/发现前解密使用；credentialID 为空时回退到该平台默认凭据。
// 明文不得写入日志，访问会留下审计记录（best-effort）。
func (s *CredentialService) Decrypted(ctx context.Context, userID, credentialID, platform string, access CredentialAccess) (map[string]string, error) {
	var cred *model.Credential
	var err error
	if credentialID == "" {
		cred, err = s.ResolveDefault(ctx, userID, platform)
	} else {
		cred, err = s.repo.GetByID(ctx, userID, credentialID)
	}
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrCredentialNotFound
		}
		if errors.Is(err, v1.ErrCredentialNotFound) {
			return nil, err
		}
		return nil, err
	}
	plain, err := s.keys.Decrypt(cred.Cipher, cred.Nonce, cred.KeyVersion)
	if err != nil {
		return nil, v1.ErrCredentialDecrypt
	}
	data := map[string]string{}
	if err := json.Unmarshal(plain, &data); err != nil {
		return nil, v1.ErrCredentialDecrypt
	}
	if err := s.auditAccess(ctx, userID, cred.Platform, cred.ID, access); err != nil {
		// 默认只告警不阻断；开启 fail-closed 后，审计写不进去就不放行凭据。
		if s.auditFailClosed {
			return nil, v1.ErrCredentialAudit
		}
	}
	return data, nil
}

func (s *CredentialService) encrypt(data map[string]string) ([]byte, []byte, int, error) {
	plain, err := json.Marshal(data)
	if err != nil {
		return nil, nil, 0, err
	}
	ciphertext, nonce, version, err := s.keys.Encrypt(plain)
	if err != nil {
		return nil, nil, 0, v1.ErrCredentialEncrypt
	}
	return ciphertext, nonce, version, nil
}

// buildPreview 生成非敏感展示串：优先标识字段脱敏，否则对令牌/密钥取头尾。
func (s *CredentialService) buildPreview(cfg *model.PlatformConfig, data map[string]string) string {
	var fields []string
	if cfg != nil && cfg.PreviewFields != "" {
		_ = json.Unmarshal([]byte(cfg.PreviewFields), &fields)
	}
	if len(fields) == 0 {
		fields = defaultPreviewFields
	}
	for _, key := range fields {
		if v := strings.TrimSpace(data[key]); v != "" {
			return maskIdentity(v)
		}
	}
	head, tail := defaultTokenHeadLen, defaultTokenTailLen
	if cfg != nil {
		if cfg.TokenHeadLen > 0 {
			head = cfg.TokenHeadLen
		}
		if cfg.TokenTailLen >= 0 {
			tail = cfg.TokenTailLen
		}
	}
	for _, key := range []string{"token", "cookie", "secret", "password"} {
		if v := strings.TrimSpace(data[key]); v != "" {
			return maskSecret(v, head, tail)
		}
	}
	for key, v := range data {
		if strings.HasPrefix(key, "__") {
			continue
		}
		if s := strings.TrimSpace(v); s != "" {
			return maskSecret(s, head, tail)
		}
	}
	return ""
}

func maskIdentity(v string) string {
	r := []rune(v)
	switch {
	case len(r) == 0:
		return ""
	case len(r) <= 2:
		return string(r[:1]) + "***"
	case len(r) < 8:
		return string(r[:1]) + "***" + string(r[len(r)-1:])
	default:
		return string(r[:3]) + "****" + string(r[len(r)-4:])
	}
}

func maskSecret(v string, head, tail int) string {
	r := []rune(v)
	if head < 1 {
		head = 1
	}
	if tail < 0 {
		tail = 0
	}
	if len(r) <= head+tail+1 {
		if len(r) <= 2 {
			return "***"
		}
		return string(r[:1]) + "***"
	}
	return string(r[:head]) + "…" + string(r[len(r)-tail:])
}

func (s *CredentialService) auditAccess(ctx context.Context, userID, platform, credentialID string, access CredentialAccess) error {
	// 审计上下文与任务 ctx 解耦，避免任务取消/超时导致漏记。
	logCtx, cancel := context.WithTimeout(context.WithoutCancel(ctx), 3*time.Second)
	defer cancel()
	now := time.Now()
	id, err := s.sid.GenString()
	if err != nil {
		s.logger.Warn("credential audit id failed", zap.Error(err))
		return err
	}
	entry := &model.CredentialAccessLog{
		ID:           id,
		UserID:       userID,
		Platform:     platform,
		CredentialID: credentialID,
		Purpose:      access.Purpose,
		RefID:        access.RefID,
		CreatedAt:    now,
	}
	if err := s.audit.Create(logCtx, entry); err != nil {
		s.logger.Warn("credential audit write failed", zap.Error(err))
		return err
	}
	if err := s.repo.MarkUsed(logCtx, credentialID, now); err != nil {
		s.logger.Warn("credential last-used update failed", zap.Error(err))
	}
	return nil
}

func toCredentialResponse(cred *model.Credential) *v1.CredentialResponse {
	return &v1.CredentialResponse{
		ID:         cred.ID,
		Platform:   cred.Platform,
		Label:      cred.Label,
		Preview:    cred.Preview,
		ProxyHint:  cred.ProxyHint,
		IsDefault:  cred.IsDefault,
		LastUsedAt: cred.LastUsedAt,
		CreatedAt:  cred.CreatedAt,
		UpdatedAt:  cred.UpdatedAt,
	}
}
