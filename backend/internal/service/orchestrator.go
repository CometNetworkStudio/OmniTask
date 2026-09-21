package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	agentv1 "backend/api/agent/v1"
	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/ws"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Orchestrator 轮询 pending 任务，解密凭据后经 gRPC 下发给 Agent，
// 汇聚事件到进度存储 / TaskEvent / WebSocket；支持取消与超时。
type Orchestrator struct {
	taskRepo *repository.TaskRepository
	creds    *CredentialService
	agent    *AgentClient
	progress *ProgressStore
	hub      *ws.Hub
	sid      *sid.Sid
	logger   *log.Logger

	sem      chan struct{}
	interval time.Duration
	timeout  time.Duration

	mu      sync.Mutex
	running map[string]context.CancelFunc

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewOrchestrator(
	taskRepo *repository.TaskRepository,
	creds *CredentialService,
	agent *AgentClient,
	progress *ProgressStore,
	hub *ws.Hub,
	sid *sid.Sid,
	conf *viper.Viper,
	logger *log.Logger,
) *Orchestrator {
	return &Orchestrator{
		taskRepo: taskRepo,
		creds:    creds,
		agent:    agent,
		progress: progress,
		hub:      hub,
		sid:      sid,
		logger:   logger,
		sem:      make(chan struct{}, 4),
		interval: 2 * time.Second,
		timeout:  conf.GetDuration("agent.task_timeout"),
		running:  map[string]context.CancelFunc{},
	}
}

func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, o.cancel = context.WithCancel(ctx)
	if n, err := o.taskRepo.MarkStaleInterrupted(ctx); err == nil && n > 0 {
		o.logger.Info("marked stale tasks interrupted", zap.Int64("count", n))
	}
	o.wg.Add(1)
	go o.loop(ctx)
	o.logger.Info("orchestrator started", zap.String("agent", "see config agent.addr"))
	return nil
}

func (o *Orchestrator) Stop(ctx context.Context) error {
	if o.cancel != nil {
		o.cancel()
	}
	o.wg.Wait()
	o.logger.Info("orchestrator stopped")
	return nil
}

func (o *Orchestrator) loop(ctx context.Context) {
	defer o.wg.Done()
	ticker := time.NewTicker(o.interval)
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-ticker.C:
			o.dispatch(ctx)
		}
	}
}

func (o *Orchestrator) dispatch(ctx context.Context) {
	tasks, err := o.taskRepo.ListByStatus(ctx, "pending", 10)
	if err != nil {
		o.logger.Error("list pending tasks", zap.Error(err))
		return
	}
	for i := range tasks {
		task := tasks[i]
		claimed, err := o.taskRepo.Claim(ctx, task.ID, "pending", "queued")
		if err != nil || !claimed {
			continue
		}
		o.wg.Add(1)
		go func() {
			defer o.wg.Done()
			o.sem <- struct{}{}
			defer func() { <-o.sem }()
			o.run(ctx, task)
		}()
	}
}

func (o *Orchestrator) register(taskID string, cancel context.CancelFunc) {
	o.mu.Lock()
	o.running[taskID] = cancel
	o.mu.Unlock()
}

func (o *Orchestrator) unregister(taskID string) {
	o.mu.Lock()
	delete(o.running, taskID)
	o.mu.Unlock()
}

func (o *Orchestrator) run(ctx context.Context, task model.Task) {
	runCtx, cancel := context.WithCancel(ctx)
	if o.timeout > 0 {
		runCtx, cancel = context.WithTimeout(ctx, o.timeout)
	}
	defer cancel()
	o.register(task.ID, cancel)
	defer o.unregister(task.ID)

	_ = o.taskRepo.UpdateStatus(ctx, task.ID, map[string]interface{}{
		"status":     "running",
		"started_at": time.Now(),
	})
	o.publish(ctx, task.ID, "running", task.Progress, "任务开始", nil)

	creds, err := o.creds.Decrypted(ctx, task.UserID, task.Platform)
	if err != nil {
		o.fail(ctx, task.ID, err.Error())
		return
	}
	params := map[string]string{}
	if task.Params != "" {
		_ = json.Unmarshal([]byte(task.Params), &params)
	}
	req := &agentv1.ExecuteRequest{
		TaskId:      task.ID,
		Platform:    task.Platform,
		Action:      task.Action,
		Params:      params,
		Credentials: creds,
	}
	terminal := false
	err = o.agent.Execute(runCtx, req, func(ev *agentv1.ExecuteEvent) {
		switch ev.GetType() {
		case agentv1.EventType_EVENT_TYPE_SUCCEEDED:
			terminal = true
			o.succeed(ctx, task.ID, ev.GetResult())
		case agentv1.EventType_EVENT_TYPE_FAILED:
			terminal = true
			o.fail(ctx, task.ID, ev.GetMessage())
		default:
			o.publish(ctx, task.ID, "running", int(ev.GetPercent()), ev.GetMessage(), ev.GetResult())
		}
	})
	if err != nil {
		switch {
		case errors.Is(runCtx.Err(), context.Canceled):
			// 取消已由 CancelTask 落库并广播，这里不覆盖状态
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			o.fail(ctx, task.ID, "任务超时")
		default:
			o.fail(ctx, task.ID, "agent 通讯失败: "+err.Error())
		}
		return
	}
	if !terminal {
		o.fail(ctx, task.ID, "agent 事件流意外结束")
	}
}

// CancelTask 取消用户自己的任务：运行中则中断子进程，排队/待执行则直接标记。
func (o *Orchestrator) CancelTask(ctx context.Context, userID, taskID string) error {
	task, err := o.taskRepo.Get(ctx, userID, taskID)
	if err != nil {
		return v1.ErrTaskNotFound
	}
	switch task.Status {
	case "succeeded", "failed", "canceled", "interrupted":
		return v1.ErrTaskNotCancelable
	}
	o.mu.Lock()
	cancel, running := o.running[taskID]
	o.mu.Unlock()

	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{
		"status":      "canceled",
		"error":       "用户取消",
		"finished_at": time.Now(),
	})
	o.publish(ctx, taskID, "canceled", task.Progress, "已取消", nil)
	if running && cancel != nil {
		cancel()
	}
	return nil
}

func (o *Orchestrator) publish(ctx context.Context, taskID, status string, percent int, message string, result map[string]string) {
	payload := map[string]interface{}{
		"task_id":   taskID,
		"status":    status,
		"percent":   percent,
		"message":   message,
		"timestamp": time.Now().Unix(),
	}
	if result != nil {
		payload["result"] = result
	}
	_ = o.progress.Set(ctx, taskID, payload)
	o.hub.Publish(taskID, payload)
	o.appendEvent(ctx, taskID, status, percent, message)
	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{"progress": percent})
}

func (o *Orchestrator) succeed(ctx context.Context, taskID string, result map[string]string) {
	if result == nil {
		result = map[string]string{}
	}
	buf, _ := json.Marshal(result)
	now := time.Now()
	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{
		"status":      "succeeded",
		"progress":    100,
		"result":      string(buf),
		"finished_at": now,
		"error":       "",
	})
	o.publish(ctx, taskID, "succeeded", 100, "任务完成", result)
}

func (o *Orchestrator) fail(ctx context.Context, taskID, message string) {
	now := time.Now()
	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{
		"status":      "failed",
		"error":       message,
		"finished_at": now,
	})
	o.publish(ctx, taskID, "failed", 0, message, nil)
}

func (o *Orchestrator) appendEvent(ctx context.Context, taskID, eventType string, percent int, message string) {
	id, err := o.sid.GenString()
	if err != nil {
		return
	}
	_ = o.taskRepo.AppendEvent(ctx, &model.TaskEvent{
		ID:      id,
		TaskID:  taskID,
		Type:    eventType,
		Percent: percent,
		Message: message,
	})
}
