package service

import (
	"context"
	"encoding/json"
	"time"

	agentv1 "backend/api/agent/v1"
	v1 "backend/api/v1"
	"backend/internal/agenthost"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
)

// DiscoverService 负责“平台条目列表”的发现、缓存（含刷新限流）与 schema 输出。
type DiscoverService struct {
	cacheRepo  *repository.PlatformCacheRepository
	scriptRepo *repository.ScriptRepository
	scripts    *ScriptService
	creds      *CredentialService
	agent      *AgentClient
	conf       *viper.Viper
	sid        *sid.Sid
	logger     *log.Logger
}

func NewDiscoverService(
	cacheRepo *repository.PlatformCacheRepository,
	scriptRepo *repository.ScriptRepository,
	scripts *ScriptService,
	creds *CredentialService,
	agent *AgentClient,
	conf *viper.Viper,
	sid *sid.Sid,
	logger *log.Logger,
) *DiscoverService {
	return &DiscoverService{cacheRepo: cacheRepo, scriptRepo: scriptRepo, scripts: scripts, creds: creds, agent: agent, conf: conf, sid: sid, logger: logger}
}

// Platforms 返回已启用脚本的完整 schema（供任意登录用户使用）。
func (s *DiscoverService) Platforms(ctx context.Context) ([]v1.PlatformResponse, error) {
	s.scripts.Sync(ctx)
	scripts, err := s.scriptRepo.List(ctx)
	if err != nil {
		return nil, err
	}
	out := make([]v1.PlatformResponse, 0, len(scripts))
	for i := range scripts {
		script := &scripts[i]
		if !script.Enabled {
			continue
		}
		out = append(out, toPlatformResponse(script))
	}
	return out, nil
}

// resolveCredentialID 指定则原样返回，否则回退到平台默认凭据（无凭据则返回空串）。
func (s *DiscoverService) resolveCredentialID(ctx context.Context, userID, platform, credentialID string) string {
	if credentialID != "" {
		return credentialID
	}
	if cred, err := s.creds.ResolveDefault(ctx, userID, platform); err == nil {
		return cred.ID
	}
	return ""
}

func (s *DiscoverService) Items(ctx context.Context, userID, platform, credentialID string) (*v1.DiscoveryResponse, error) {
	s.scripts.Sync(ctx)
	manifest := s.manifest(ctx, platform)
	resp := &v1.DiscoveryResponse{Platform: platform}
	if manifest != nil && manifest.Discovery != nil {
		resp.Label = manifest.Discovery.Label
		resp.Actions = manifest.Discovery.Actions
		for _, col := range manifest.Discovery.Columns {
			resp.Columns = append(resp.Columns, toPlatformField(col))
		}
	}
	cache, err := s.cacheRepo.Get(ctx, userID, platform, s.resolveCredentialID(ctx, userID, platform, credentialID))
	if err == nil {
		_ = json.Unmarshal([]byte(cache.Items), &resp.Items)
		refreshed := cache.RefreshedAt
		resp.RefreshedAt = &refreshed
	}
	if resp.Items == nil {
		resp.Items = []v1.DiscoveryItem{}
	}
	return resp, nil
}

// Refresh 触发一次发现（受 discovery.min_interval 限流），并写入缓存。
// params 非空时视为带筛选的临时查询，不读写缓存。
func (s *DiscoverService) Refresh(ctx context.Context, userID, platform, credentialID string, params map[string]string) (*v1.DiscoveryResponse, error) {
	s.scripts.Sync(ctx)
	manifest := s.manifest(ctx, platform)
	if manifest == nil {
		return nil, v1.ErrUnsupportedPlatform
	}
	credID := s.resolveCredentialID(ctx, userID, platform, credentialID)
	filtered := len(params) > 0
	if !filtered {
		interval := s.conf.GetInt("discovery.min_interval")
		if interval <= 0 {
			interval = 60
		}
		if cache, err := s.cacheRepo.Get(ctx, userID, platform, credID); err == nil {
			if time.Since(cache.RefreshedAt) < time.Duration(interval)*time.Second {
				return s.Items(ctx, userID, platform, credID) // 距上次刷新过近，直接返回缓存
			}
		}
	}

	creds, err := s.creds.Decrypted(ctx, userID, credID, platform, CredentialAccess{Purpose: "discover"})
	if err != nil {
		return nil, err
	}
	proxy := creds[proxyKey]
	delete(creds, proxyKey)
	result, err := s.agent.Discover(ctx, &agentv1.DiscoverRequest{Platform: platform, Params: params, Credentials: creds, Proxy: proxy})
	if err != nil {
		return nil, err
	}
	items := make([]v1.DiscoveryItem, 0, len(result.GetItems()))
	for _, item := range result.GetItems() {
		items = append(items, v1.DiscoveryItem{Key: item.GetKey(), Fields: item.GetFields()})
	}
	now := time.Now()
	if filtered {
		resp := schemaResponse(platform, manifest)
		resp.Items = items
		resp.RefreshedAt = &now
		return resp, nil
	}
	buf, _ := json.Marshal(items)
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	if err := s.cacheRepo.Upsert(ctx, &model.PlatformCache{
		ID: id, UserID: userID, Platform: platform, CredentialID: credID, Items: string(buf),
		RefreshedAt: now, UpdatedAt: now,
	}); err != nil {
		return nil, err
	}
	return s.Items(ctx, userID, platform, credID)
}

func schemaResponse(platform string, manifest *agenthost.Manifest) *v1.DiscoveryResponse {
	resp := &v1.DiscoveryResponse{Platform: platform, Items: []v1.DiscoveryItem{}}
	if manifest != nil && manifest.Discovery != nil {
		resp.Label = manifest.Discovery.Label
		resp.Actions = manifest.Discovery.Actions
		for _, col := range manifest.Discovery.Columns {
			resp.Columns = append(resp.Columns, toPlatformField(col))
		}
	}
	return resp
}

func (s *DiscoverService) manifest(ctx context.Context, platform string) *agenthost.Manifest {
	script, err := s.scriptRepo.Get(ctx, platform)
	if err != nil || !script.Enabled {
		return nil
	}
	var m agenthost.Manifest
	if script.Manifest != "" && json.Unmarshal([]byte(script.Manifest), &m) == nil {
		return &m
	}
	return nil
}

func toPlatformResponse(script *model.Script) v1.PlatformResponse {
	resp := v1.PlatformResponse{
		ID: script.ID, Name: script.Name, Version: script.Version,
		Status: script.Status, Actions: []v1.PlatformAction{},
		Params: []v1.PlatformField{}, Credentials: []v1.PlatformField{},
	}
	var m agenthost.Manifest
	if script.Manifest == "" || json.Unmarshal([]byte(script.Manifest), &m) != nil {
		return resp
	}
	resp.Runtime = m.Runtime
	resp.Icon = m.Icon
	for _, a := range m.Actions {
		action := v1.PlatformAction{ID: a.ID, Name: a.Name, Description: a.Description, Params: []v1.PlatformField{}}
		for _, p := range a.Params {
			action.Params = append(action.Params, toPlatformField(p))
		}
		if a.Result != nil {
			action.Result = &v1.PlatformResult{Type: a.Result.Type}
			for _, col := range a.Result.Columns {
				action.Result.Columns = append(action.Result.Columns, toPlatformField(col))
			}
		}
		resp.Actions = append(resp.Actions, action)
	}
	for _, p := range m.Params {
		resp.Params = append(resp.Params, toPlatformField(p))
	}
	for _, c := range m.Credentials {
		resp.Credentials = append(resp.Credentials, toPlatformField(c))
	}
	if m.Discovery != nil {
		disc := &v1.PlatformDiscovery{Label: m.Discovery.Label, Actions: m.Discovery.Actions, ParamMap: m.Discovery.ParamMap, Primary: m.Discovery.Primary}
		for _, col := range m.Discovery.Columns {
			disc.Columns = append(disc.Columns, toPlatformField(col))
		}
		resp.Discovery = disc
	}
	return resp
}

func toPlatformField(f agenthost.Field) v1.PlatformField {
	out := v1.PlatformField{
		Key: f.Key, Label: f.Label, Type: f.Type, Default: f.Default,
		Required: f.Required, Placeholder: f.Placeholder, Help: f.Help,
		Min: f.Min, Max: f.Max, Step: f.Step, Multiline: f.Multiline,
	}
	for _, opt := range f.Options {
		out.Options = append(out.Options, v1.PlatformOption{Label: opt.Label, Value: opt.Value})
	}
	return out
}
