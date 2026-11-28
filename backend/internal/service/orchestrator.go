package service

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	hostv1 "backend/api/host/v1"
	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/internal/ws"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"go.uber.org/zap"
)

// Orchestrator 轮询 pending 任务，解密凭据后经 gRPC 下发给 Host，
// 汇聚事件到进度存储 / TaskEvent / WebSocket；支持取消与超时。
type Orchestrator struct {
	taskRepo *repository.TaskRepository
	creds    *CredentialService
	host     *HostClient
	progress *ProgressStore
	hub      *ws.Hub
	sid      *sid.Sid
	logger   *log.Logger

	global        chan struct{}
	platformLimit int
	platformSems  map[string]chan struct{}
	accountSems   map[string]chan struct{}
	semMu         sync.Mutex
	interval      time.Duration
	timeout       time.Duration

	mu      sync.Mutex
	running map[string]context.CancelFunc

	cancel context.CancelFunc
	wg     sync.WaitGroup
}

func NewOrchestrator(
	taskRepo *repository.TaskRepository,
	creds *CredentialService,
	host *HostClient,
	progress *ProgressStore,
	hub *ws.Hub,
	sid *sid.Sid,
	conf *viper.Viper,
	logger *log.Logger,
) *Orchestrator {
	globalLimit := conf.GetInt("host.global_concurrency")
	if globalLimit <= 0 {
		globalLimit = 4
	}
	platformLimit := conf.GetInt("host.platform_concurrency")
	if platformLimit <= 0 {
		platformLimit = 2
	}
	return &Orchestrator{
		taskRepo:      taskRepo,
		creds:         creds,
		host:          host,
		progress:      progress,
		hub:           hub,
		sid:           sid,
		logger:        logger,
		global:        make(chan struct{}, globalLimit),
		platformLimit: platformLimit,
		platformSems:  map[string]chan struct{}{},
		accountSems:   map[string]chan struct{}{},
		interval:      2 * time.Second,
		timeout:       conf.GetDuration("host.task_timeout"),
		running:       map[string]context.CancelFunc{},
	}
}

func (o *Orchestrator) platformSem(platform string) chan struct{} {
	o.semMu.Lock()
	defer o.semMu.Unlock()
	sem, ok := o.platformSems[platform]
	if !ok {
		sem = make(chan struct{}, o.platformLimit)
		o.platformSems[platform] = sem
	}
	return sem
}

func (o *Orchestrator) accountSem(key string) chan struct{} {
	o.semMu.Lock()
	defer o.semMu.Unlock()
	sem, ok := o.accountSems[key]
	if !ok {
		sem = make(chan struct{}, 1)
		o.accountSems[key] = sem
	}
	return sem
}

// acquire 依次获取全局、平台、账号三级并发额度；ctx 取消时返回 false。
func (o *Orchestrator) acquire(ctx context.Context, task model.Task) (func(), bool) {
	select {
	case o.global <- struct{}{}:
	case <-ctx.Done():
		return nil, false
	}
	platform := o.platformSem(task.Platform)
	select {
	case platform <- struct{}{}:
	case <-ctx.Done():
		<-o.global
		return nil, false
	}
	accountKey := task.UserID + ":" + task.CredentialID
	account := o.accountSem(accountKey)
	select {
	case account <- struct{}{}:
	case <-ctx.Done():
		<-platform
		<-o.global
		return nil, false
	}
	return func() {
		<-account
		<-platform
		<-o.global
	}, true
}

func (o *Orchestrator) Start(ctx context.Context) error {
	ctx, o.cancel = context.WithCancel(ctx)
	if n, err := o.taskRepo.MarkStaleInterrupted(ctx); err == nil && n > 0 {
		o.logger.Info("marked stale tasks interrupted", zap.Int64("count", n))
	}
	o.wg.Add(1)
	go o.loop(ctx)
	o.logger.Info("orchestrator started", zap.String("host", "see config host.addr"))
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
	release, ok := o.acquire(runCtx, task)
	if !ok {
		return
	}
	defer release()

	_ = o.taskRepo.UpdateStatus(ctx, task.ID, map[string]interface{}{
		"status":     "running",
		"started_at": time.Now(),
	})
	lastPercent := task.Progress
	o.publish(ctx, task.ID, "running", lastPercent, "任务开始", nil, eventMeta{Kind: "started"})

	creds := map[string]string{}
	proxy := ""
	if task.CredentialID != "" {
		var err error
		creds, err = o.creds.Decrypted(ctx, task.UserID, task.CredentialID, task.Platform, CredentialAccess{Purpose: "task", RefID: task.ID})
		if err != nil {
			o.fail(ctx, task.ID, err.Error(), lastPercent)
			return
		}
		proxy = creds[proxyKey]
		delete(creds, proxyKey)
	}
	params := map[string]string{}
	if task.Params != "" {
		_ = json.Unmarshal([]byte(task.Params), &params)
	}
	req := &hostv1.ExecuteRequest{
		TaskId:      task.ID,
		Platform:    task.Platform,
		Action:      task.Action,
		Params:      params,
		Credentials: creds,
		Proxy:       proxy,
	}
	terminal := false
	runErr := o.host.Execute(runCtx, req, func(ev *hostv1.ExecuteEvent) {
		if runCtx.Err() != nil {
			return
		}
		switch ev.GetType() {
		case hostv1.EventType_EVENT_TYPE_SUCCEEDED:
			terminal = true
			o.succeed(ctx, task.ID, ev.GetResult(), ev.GetResultJson())
		case hostv1.EventType_EVENT_TYPE_FAILED:
			terminal = true
			o.fail(ctx, task.ID, ev.GetMessage(), lastPercent)
		case hostv1.EventType_EVENT_TYPE_STARTED:
			// 任务开始时后端已落一条 started，脚本侧 started 只用于本机调试。
		case hostv1.EventType_EVENT_TYPE_LOG:
			o.publish(ctx, task.ID, "running", lastPercent, ev.GetMessage(), nil, eventMeta{
				Kind: "log", Level: ev.GetLevel(),
			})
		case hostv1.EventType_EVENT_TYPE_STEP:
			o.publish(ctx, task.ID, "running", lastPercent, ev.GetMessage(), nil, eventMeta{
				Kind: "step", Step: ev.GetStep(), Label: ev.GetLabel(), StepStatus: ev.GetStatus(),
			})
		case hostv1.EventType_EVENT_TYPE_METRIC:
			o.publish(ctx, task.ID, "running", lastPercent, ev.GetMessage(), nil, eventMeta{
				Kind: "metric", Key: ev.GetKey(), Label: ev.GetLabel(), Value: ev.GetValue(), Unit: ev.GetUnit(),
			})
		case hostv1.EventType_EVENT_TYPE_HEARTBEAT:
			// 心跳只用于保活，不落事件表。
		default:
			if ev.GetHasPercent() {
				lastPercent = int(ev.GetPercent())
			}
			o.publish(ctx, task.ID, "running", lastPercent, ev.GetMessage(), ev.GetResult(), eventMeta{
				Kind: "progress", HasPercent: ev.GetHasPercent(),
				Current: int(ev.GetCurrent()), Total: int(ev.GetTotal()),
				Unit: ev.GetUnit(), Stage: ev.GetStage(),
			})
		}
	})
	if runErr != nil {
		switch {
		case errors.Is(runCtx.Err(), context.Canceled):
			// 取消已由 CancelTask 落库并广播，这里不覆盖状态
		case errors.Is(runCtx.Err(), context.DeadlineExceeded):
			o.fail(ctx, task.ID, "任务超时", lastPercent)
		default:
			o.fail(ctx, task.ID, "Host 通讯失败: "+runErr.Error(), lastPercent)
		}
		return
	}
	if !terminal {
		o.fail(ctx, task.ID, "Host 事件流意外结束", lastPercent)
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
	o.publish(ctx, taskID, "canceled", task.Progress, "已取消", nil, eventMeta{Kind: "canceled"})
	if running && cancel != nil {
		cancel()
	}
	return nil
}

type eventMeta struct {
	Kind       string
	Level      string
	Step       string
	Label      string
	StepStatus string
	Current    int
	Total      int
	Unit       string
	Stage      string
	Key        string
	Value      string
	HasPercent bool
}

func (o *Orchestrator) publish(ctx context.Context, taskID, status string, percent int, message string, result map[string]string, meta eventMeta) {
	if meta.Kind == "" {
		meta.Kind = "progress"
	}
	eventID := o.appendEvent(ctx, taskID, status, percent, message, result, meta)
	payload := map[string]interface{}{
		"id":          eventID,
		"task_id":     taskID,
		"status":      status,
		"kind":        meta.Kind,
		"percent":     percent,
		"message":     message,
		"level":       meta.Level,
		"step":        meta.Step,
		"label":       meta.Label,
		"step_status": meta.StepStatus,
		"current":     meta.Current,
		"total":       meta.Total,
		"unit":        meta.Unit,
		"stage":       meta.Stage,
		"key":         meta.Key,
		"value":       meta.Value,
		"timestamp":   time.Now().Unix(),
	}
	if result != nil {
		payload["result"] = result
	}
	_ = o.progress.Set(ctx, taskID, payload)
	o.hub.Publish(taskID, payload)
	if meta.HasPercent {
		_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{"progress": percent})
	}
}

func (o *Orchestrator) succeed(ctx context.Context, taskID string, result map[string]string, resultJSON string) {
	if result == nil {
		result = map[string]string{}
	}
	stored := resultJSON
	if stored == "" || !json.Valid([]byte(stored)) {
		buf, _ := json.Marshal(result)
		stored = string(buf)
	}
	now := time.Now()
	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{
		"status":      "succeeded",
		"progress":    100,
		"result":      stored,
		"finished_at": now,
		"error":       "",
	})
	o.publish(ctx, taskID, "succeeded", 100, "任务完成", result, eventMeta{Kind: "succeeded", HasPercent: true})
}

func (o *Orchestrator) fail(ctx context.Context, taskID, message string, percent int) {
	now := time.Now()
	_ = o.taskRepo.UpdateStatus(ctx, taskID, map[string]interface{}{
		"status":      "failed",
		"error":       message,
		"finished_at": now,
	})
	o.publish(ctx, taskID, "failed", percent, message, nil, eventMeta{Kind: "failed"})
}

func (o *Orchestrator) appendEvent(ctx context.Context, taskID, status string, percent int, message string, result map[string]string, meta eventMeta) string {
	id, err := o.sid.GenString()
	if err != nil {
		return ""
	}
	_ = o.taskRepo.AppendEvent(ctx, &model.TaskEvent{
		ID:         id,
		TaskID:     taskID,
		Type:       status,
		Kind:       meta.Kind,
		Percent:    percent,
		Message:    message,
		Level:      meta.Level,
		Step:       meta.Step,
		Label:      meta.Label,
		StepStatus: meta.StepStatus,
		Current:    meta.Current,
		Total:      meta.Total,
		Unit:       meta.Unit,
		Stage:      meta.Stage,
		Key:        meta.Key,
		Value:      meta.Value,
	})
	return id
}
