package task

import (
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
)

type Task struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     repository.Transaction
}

func NewTask(
	tm repository.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Task {
	return &Task{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
