package service

import (
	"context"
	"encoding/json"
	"time"

	"github.com/redis/go-redis/v9"
)

// ProgressStore 把任务最新进度写入 Redis，供 WebSocket 首帧回放与状态查询。
type ProgressStore struct {
	rdb *redis.Client
}

func NewProgressStore(rdb *redis.Client) *ProgressStore {
	return &ProgressStore{rdb: rdb}
}

func (p *ProgressStore) key(taskID string) string {
	return "omnitask:task:progress:" + taskID
}

func (p *ProgressStore) Set(ctx context.Context, taskID string, payload map[string]interface{}) error {
	buf, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return p.rdb.Set(ctx, p.key(taskID), buf, time.Hour).Err()
}

func (p *ProgressStore) Get(ctx context.Context, taskID string) (map[string]interface{}, bool) {
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
