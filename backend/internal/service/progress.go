package service

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
)

// ProgressStore 保存任务最新进度：full 模式写 Redis，lite 模式写进程内。
type ProgressStore struct {
	rdb *redis.Client

	mu  sync.RWMutex
	mem map[string]map[string]interface{}
}

func NewProgressStore(rdb *redis.Client) *ProgressStore {
	return &ProgressStore{rdb: rdb, mem: map[string]map[string]interface{}{}}
}

func (p *ProgressStore) key(taskID string) string {
	return "omnitask:task:progress:" + taskID
}

func (p *ProgressStore) Set(ctx context.Context, taskID string, payload map[string]interface{}) error {
	if p.rdb == nil {
		p.mu.Lock()
		p.mem[taskID] = payload
		p.mu.Unlock()
		return nil
	}
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rdb.Set(ctx, p.key(taskID), buf, time.Hour).Err()
}

func (p *ProgressStore) Get(ctx context.Context, taskID string) (map[string]interface{}, bool) {
	if p.rdb == nil {
		p.mu.RLock()
		payload, ok := p.mem[taskID]
		p.mu.RUnlock()
		return payload, ok
	}
	buf, err := p.rdb.Get(ctx, p.key(taskID)).Bytes()
	if err != nil {
		return nil, false
	}
	var out map[string]interface{}
	if err := json.Unmarshal(buf, &out); err != nil {
		return nil, false
	}
	return out, true
}
