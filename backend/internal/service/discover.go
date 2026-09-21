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
	creds      *CredentialService
	agent      *AgentClient
	conf       *viper.Viper
	sid        *sid.Sid
	logger     *log.Logger
}

func NewDiscoverService(
	cacheRepo *repository.PlatformCacheRepository,
	scriptRepo *repository.ScriptRepository,
	creds *CredentialService,
	agent *AgentClient,
	conf *viper.Viper,
	sid *sid.Sid,
	logger *log.Logger,
) *DiscoverService {
	return &DiscoverService{cacheRepo: cacheRepo, scriptRepo: scriptRepo, creds: creds, agent: agent, conf: conf, sid: sid, logger: logger}
}

// Platforms 返回已启用脚本的完整 schema（供任意登录用户使用）。
func (s *DiscoverService) Platforms(ctx context.Context) ([]v1.PlatformResponse, error) {
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

func (s *DiscoverService) Items(ctx context.Context, userID, platform string) (*v1.DiscoveryResponse, error) {
	manifest := s.manifest(ctx, platform)
	resp := &v1.DiscoveryResponse{Platform: platform}
	if manifest != nil && manifest.Discovery != nil {
		resp.Label = manifest.Discovery.Label
		resp.Actions = manifest.Discovery.Actions
		for _, col := range manifest.Discovery.Columns {
			resp.Columns = append(resp.Columns, toPlatformField(col))
		}
	}
	cache, err := s.cacheRepo.Get(ctx, userID, platform)
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
func (s *DiscoverService) Refresh(ctx context.Context, userID, platform string) (*v1.DiscoveryResponse, error) {
	manifest := s.manifest(ctx, platform)
	if manifest == nil {
		return nil, v1.ErrUnsupportedPlatform
	}
	interval := s.conf.GetInt("discovery.min_interval")
	if interval <= 0 {
		interval = 60
	}
	if cache, err := s.cacheRepo.Get(ctx, userID, platform); err == nil {
		if time.Since(cache.RefreshedAt) < time.Duration(interval)*time.Second {
			return s.Items(ctx, userID, platform) // 距上次刷新过近，直接返回缓存
		}
	}

	creds, err := s.creds.Decrypted(ctx, userID, platform)
	if err != nil {
		return nil, err
	}
	result, err := s.agent.Discover(ctx, &agentv1.DiscoverRequest{Platform: platform, Credentials: creds})
	if err != nil {
		return nil, err
	}
	items := make([]v1.DiscoveryItem, 0, len(result.GetItems()))
	for _, item := range result.GetItems() {
		items = append(items, v1.DiscoveryItem{Key: item.GetKey(), Fields: item.GetFields()})
	}
	buf, _ := json.Marshal(items)
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	now := time.Now()
	if err := s.cacheRepo.Upsert(ctx, &model.PlatformCache{
		ID: id, UserID: userID, Platform: platform, Items: string(buf),
		RefreshedAt: now, UpdatedAt: now,
	}); err != nil {
		return nil, err
	}
	return s.Items(ctx, userID, platform)
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
	for _, a := range m.Actions {
		resp.Actions = append(resp.Actions, v1.PlatformAction{ID: a.ID, Name: a.Name})
	}
	for _, p := range m.Params {
		resp.Params = append(resp.Params, toPlatformField(p))
	}
	for _, c := range m.Credentials {
		resp.Credentials = append(resp.Credentials, toPlatformField(c))
	}
	if m.Discovery != nil {
		disc := &v1.PlatformDiscovery{Label: m.Discovery.Label, Actions: m.Discovery.Actions, ParamMap: m.Discovery.ParamMap}
		for _, col := range m.Discovery.Columns {
			disc.Columns = append(disc.Columns, toPlatformField(col))
		}
		resp.Discovery = disc
	}
	return resp
}

func toPlatformField(f agenthost.Field) v1.PlatformField {
	return v1.PlatformField{Key: f.Key, Label: f.Label, Type: f.Type, Default: f.Default}
}
