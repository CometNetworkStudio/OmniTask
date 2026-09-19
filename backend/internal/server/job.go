package server

import (
	"backend/pkg/log"
	"context"
)

type JobServer struct {
	log *log.Logger
}

func NewJobServer(
	log *log.Logger,
) *JobServer {
	return &JobServer{
		log: log,
	}
}

func (j *JobServer) Start(ctx context.Context) error {
	<-ctx.Done()
	return nil
}
func (j *JobServer) Stop(ctx context.Context) error {
	return nil
}
