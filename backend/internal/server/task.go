package server

import (
	"context"
	"github.com/go-co-op/gocron"
	"backend/pkg/log"
	"go.uber.org/zap"
	"time"
)

type TaskServer struct {
	log       *log.Logger
	scheduler *gocron.Scheduler
}

func NewTaskServer(
	log *log.Logger,
) *TaskServer {
	return &TaskServer{
		log: log,
	}
}
func (t *TaskServer) Start(ctx context.Context) error {
	gocron.SetPanicHandler(func(jobName string, recoverData interface{}) {
		t.log.Error("TaskServer Panic", zap.String("job", jobName), zap.Any("recover", recoverData))
	})

	t.scheduler = gocron.NewScheduler(time.UTC)
	t.scheduler.StartBlocking()
	return nil
}
func (t *TaskServer) Stop(ctx context.Context) error {
	t.scheduler.Stop()
	t.log.Info("TaskServer stop...")
	return nil
}
