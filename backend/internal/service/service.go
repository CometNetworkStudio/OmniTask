package service

import (
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
)

type Service struct {
	logger *log.Logger
	sid    *sid.Sid
	tm     repository.Transaction
}

func NewService(
	tm repository.Transaction,
	logger *log.Logger,
	sid *sid.Sid,
) *Service {
	return &Service{
		logger: logger,
		sid:    sid,
		tm:     tm,
	}
}
