package job

import (
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
)

type Job struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     repository.Transaction
}

func NewJob(
	tm repository.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Job {
	return &Job{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
