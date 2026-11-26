package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"time"

	v1 "backend/api/v1"
	"backend/internal/agenthost"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"github.com/spf13/viper"
	"gorm.io/gorm"
)

var SupportedPlatforms = map[string]bool{"cidaren": true, "welearn": true}

func IsPlatformSupported(platform string) bool { return SupportedPlatforms[platform] }

type TaskService struct {
	repo       *repository.TaskRepository
	scriptRepo *repository.ScriptRepository
	scripts    *ScriptService
	creds      *CredentialService
	sid        *sid.Sid
	conf       *viper.Viper
	logger     *log.Logger
}

func NewTaskService(repo *repository.TaskRepository, scriptRepo *repository.ScriptRepository, scripts *ScriptService, creds *CredentialService, sid *sid.Sid, conf *viper.Viper, logger *log.Logger) *TaskService {
	return &TaskService{repo: repo, scriptRepo: scriptRepo, scripts: scripts, creds: creds, sid: sid, conf: conf, logger: logger}
}

func (s *TaskService) allowedPlatform(ctx context.Context, platform string) bool {
	if IsPlatformSupported(platform) {
		return true
	}
	ids, err := s.scriptRepo.EnabledIDs(ctx)
	if err != nil {
		return false
	}
	for _, id := range ids {
		if id == platform {
			return true
		}
	}
	return false
}

// validateTask 用已安装脚本的 manifest 做动作与必填参数校验；无 manifest 时放行。
func (s *TaskService) validateTask(ctx context.Context, req *v1.TaskCreateRequest) error {
	script, err := s.scriptRepo.Get(ctx, req.Platform)
	if err != nil || script.Manifest == "" {
		return nil
	}
	var m agenthost.Manifest
	if err := json.Unmarshal([]byte(script.Manifest), &m); err != nil {
		return nil
	}
	if len(m.Actions) == 0 {
		return nil
	}
	var target *agenthost.Action
	for i := range m.Actions {
		if m.Actions[i].ID == req.Action {
			target = &m.Actions[i]
			break
		}
	}
	if target == nil {
		return fmt.Errorf("%w: 动作不存在 %q", v1.ErrInvalidTaskParams, req.Action)
	}
	fields := target.Params
	if len(fields) == 0 {
		fields = m.Params
	}
	for _, field := range fields {
		if field.Required && strings.TrimSpace(req.Params[field.Key]) == "" {
			return fmt.Errorf("%w: 缺少必填参数 %s", v1.ErrInvalidTaskParams, field.Label)
		}
	}
	return nil
}

func (s *TaskService) Create(ctx context.Context, userID string, req *v1.TaskCreateRequest) (*v1.TaskResponse, error) {
	if s.scripts != nil {
		s.scripts.Sync(ctx)
	}
	if !s.allowedPlatform(ctx, req.Platform) {
		return nil, v1.ErrUnsupportedPlatform
	}
	if limit := s.conf.GetInt("task.max_active_per_user"); limit > 0 {
		count, err := s.repo.CountActiveByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if count >= int64(limit) {
			return nil, v1.ErrTaskLimit
		}
	}
	if limit := s.conf.GetInt("task.daily_tasks_per_user"); limit > 0 {
		now := time.Now()
		start := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
		count, err := s.repo.CountCreatedSince(ctx, userID, start)
		if err != nil {
			return nil, err
		}
		if count >= int64(limit) {
			return nil, v1.ErrDailyTaskLimit
		}
	}
	if limit := s.conf.GetInt("task.max_pending_per_user"); limit > 0 {
		count, err := s.repo.CountPendingByUser(ctx, userID)
		if err != nil {
			return nil, err
		}
		if count >= int64(limit) {
			return nil, v1.ErrPendingTaskLimit
		}
	}
	if limit := s.conf.GetInt("task.max_pending"); limit > 0 {
		count, err := s.repo.CountByStatuses(ctx, "pending", "queued")
		if err != nil {
			return nil, err
		}
		if count >= int64(limit) {
			return nil, v1.ErrQueueFull
		}
	}
	if err := s.validateTask(ctx, req); err != nil {
		return nil, err
	}
	params, err := json.Marshal(req.Params)
	if err != nil {
		return nil, err
	}
	paramsStr := string(params)
	if paramsStr == "" || paramsStr == "null" {
		paramsStr = "{}"
	}
	// 解析使用的凭据：显式指定则校验归属，否则用平台默认；都没有则留空由脚本自行处理。
	credentialID := req.CredentialID
	if credentialID != "" {
		if _, err := s.creds.GetOwned(ctx, userID, credentialID); err != nil {
			return nil, err
		}
	} else if cred, err := s.creds.ResolveDefault(ctx, userID, req.Platform); err == nil {
		credentialID = cred.ID
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	task := &model.Task{
		ID:           id,
		UserID:       userID,
		Platform:     req.Platform,
		Action:       req.Action,
		CredentialID: credentialID,
		Params:       paramsStr,
		Status:       "pending",
		Result:       "{}",
	}
	if err := s.repo.Create(ctx, task); err != nil {
		return nil, err
	}
	return toTaskResponse(task), nil
}

func (s *TaskService) Get(ctx context.Context, userID, id string) (*v1.TaskResponse, error) {
	task, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrTaskNotFound
		}
		return nil, err
	}
	return toTaskResponse(task), nil
}

// Retry 重置失败/取消/中断的任务为待执行，重新排队。
func (s *TaskService) Retry(ctx context.Context, userID, id string) (*v1.TaskResponse, error) {
	task, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrTaskNotFound
		}
		return nil, err
	}
	switch task.Status {
	case "failed", "canceled", "interrupted":
	default:
		return nil, v1.ErrTaskNotCancelable
	}
	if err := s.repo.DeleteEvents(ctx, id); err != nil {
		return nil, err
	}
	if err := s.repo.Reset(ctx, userID, id); err != nil {
		return nil, err
	}
	return s.Get(ctx, userID, id)
}

// Delete 删除任务及其事件（运行中/排队中需先取消）。
func (s *TaskService) Delete(ctx context.Context, userID, id string) error {
	task, err := s.repo.Get(ctx, userID, id)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return v1.ErrTaskNotFound
		}
		return err
	}
	if task.Status == "running" || task.Status == "queued" {
		return v1.ErrTaskNotCancelable
	}
	if _, err := s.repo.Delete(ctx, userID, id); err != nil {
		return err
	}
	if err := s.repo.DeleteEvents(ctx, id); err != nil {
		return err
	}
	// 删除任务后，若某归档凭据已无任何任务，则真正清除。
	s.creds.PurgeArchived(ctx, userID, task.Platform)
	return nil
}

// Events 返回任务的历史展示事件，供详情页刷新/回放。
func (s *TaskService) Events(ctx context.Context, userID, id string, limit int) ([]v1.TaskEventResponse, error) {
	if _, err := s.repo.Get(ctx, userID, id); err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, v1.ErrTaskNotFound
		}
		return nil, err
	}
	events, err := s.repo.ListEvents(ctx, id, limit)
	if err != nil {
		return nil, err
	}
	out := make([]v1.TaskEventResponse, 0, len(events))
	for _, e := range events {
		out = append(out, v1.TaskEventResponse{
			ID: e.ID, TaskID: e.TaskID, Type: e.Type, Kind: e.Kind, Percent: e.Percent,
			Message: e.Message, Level: e.Level, Step: e.Step, Label: e.Label,
			StepStatus: e.StepStatus, Current: e.Current, Total: e.Total, Unit: e.Unit,
			Stage: e.Stage, Key: e.Key, Value: e.Value, CreatedAt: e.CreatedAt,
		})
	}
	return out, nil
}

func (s *TaskService) List(ctx context.Context, userID, status, platform string, page, size int) ([]v1.TaskResponse, int64, error) {
	page, size = normalizePage(s.conf, page, size)
	tasks, total, err := s.repo.ListByUser(ctx, userID, status, platform, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]v1.TaskResponse, 0, len(tasks))
	for i := range tasks {
		out = append(out, *toTaskResponse(&tasks[i]))
	}
	return out, total, nil
}

func toTaskResponse(task *model.Task) *v1.TaskResponse {
	resp := &v1.TaskResponse{
		ID:           task.ID,
		Platform:     task.Platform,
		Action:       task.Action,
		CredentialID: task.CredentialID,
		Status:       task.Status,
		Progress:     task.Progress,
		Error:        task.Error,
		StartedAt:    task.StartedAt,
		FinishedAt:   task.FinishedAt,
		CreatedAt:    task.CreatedAt,
		UpdatedAt:    task.UpdatedAt,
		Params:       map[string]string{},
	}
	if task.Params != "" {
		_ = json.Unmarshal([]byte(task.Params), &resp.Params)
	}
	if task.Result != "" {
		_ = json.Unmarshal([]byte(task.Result), &resp.Result)
	}
	if resp.Result == nil {
		resp.Result = map[string]any{}
	}
	return resp
}
