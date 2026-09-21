package service

import (
	"context"
	"encoding/json"
	"errors"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
	"gorm.io/gorm"
)

var SupportedPlatforms = map[string]bool{"cidaren": true, "welearn": true}

func IsPlatformSupported(platform string) bool { return SupportedPlatforms[platform] }

type TaskService struct {
	repo       *repository.TaskRepository
	scriptRepo *repository.ScriptRepository
	sid        *sid.Sid
	logger     *log.Logger
}

func NewTaskService(repo *repository.TaskRepository, scriptRepo *repository.ScriptRepository, sid *sid.Sid, logger *log.Logger) *TaskService {
	return &TaskService{repo: repo, scriptRepo: scriptRepo, sid: sid, logger: logger}
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

func (s *TaskService) Create(ctx context.Context, userID string, req *v1.TaskCreateRequest) (*v1.TaskResponse, error) {
	if !s.allowedPlatform(ctx, req.Platform) {
		return nil, v1.ErrUnsupportedPlatform
	}
	params, err := json.Marshal(req.Params)
	if err != nil {
		return nil, err
	}
	paramsStr := string(params)
	if paramsStr == "" || paramsStr == "null" {
		paramsStr = "{}"
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	task := &model.Task{
		ID:       id,
		UserID:   userID,
		Platform: req.Platform,
		Action:   req.Action,
		Params:   paramsStr,
		Status:   "pending",
		Result:   "{}",
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
	return s.repo.DeleteEvents(ctx, id)
}

func (s *TaskService) List(ctx context.Context, userID string, page, size int) ([]v1.TaskResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	tasks, total, err := s.repo.ListByUser(ctx, userID, size, (page-1)*size)
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
		ID:         task.ID,
		Platform:   task.Platform,
		Action:     task.Action,
		Status:     task.Status,
		Progress:   task.Progress,
		Error:      task.Error,
		StartedAt:  task.StartedAt,
		FinishedAt: task.FinishedAt,
		CreatedAt:  task.CreatedAt,
		UpdatedAt:  task.UpdatedAt,
		Params:     map[string]string{},
		Result:     map[string]string{},
	}
	if task.Params != "" {
		_ = json.Unmarshal([]byte(task.Params), &resp.Params)
	}
	if task.Result != "" {
		_ = json.Unmarshal([]byte(task.Result), &resp.Result)
	}
	return resp
}
