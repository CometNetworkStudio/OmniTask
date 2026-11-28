package server

import (
	"context"
	"errors"

	hostv1 "backend/api/host/v1"
	apiv1 "backend/api/v1"
	"backend/internal/service"
	"backend/pkg/log"
	grpcserver "backend/pkg/server/grpc"
	"github.com/spf13/viper"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
	"gorm.io/gorm"
)

// bankServer 实现 proto 的 QuestionBank 服务，供 Python 脚本 Host 读写题库。
type bankServer struct {
	hostv1.UnimplementedQuestionBankServer
	svc    *service.QuestionService
	logger *log.Logger
}

func NewBankGRPCServer(svc *service.QuestionService, conf *viper.Viper, logger *log.Logger) *grpcserver.Server {
	s := grpcserver.NewServer(
		logger,
		grpcserver.WithServerHost(conf.GetString("grpc.host")),
		grpcserver.WithServerPort(conf.GetInt("grpc.port")),
	)
	hostv1.RegisterQuestionBankServer(s.Server, &bankServer{svc: svc, logger: logger})
	return s
}

func (b *bankServer) Lookup(ctx context.Context, req *hostv1.LookupRequest) (*hostv1.LookupResponse, error) {
	q, err := b.svc.Lookup(ctx, req.GetPlatform(), req.GetQuestionKey())
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return &hostv1.LookupResponse{Found: false}, nil
		}
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &hostv1.LookupResponse{
		Found: true,
		Question: &hostv1.Question{
			Platform:    q.Platform,
			QuestionKey: q.QuestionKey,
			Stem:        q.Stem,
			Options:     q.Options,
			Answer:      q.Answer,
			Source:      q.Source,
			Confidence:  q.Confidence,
			HitCount:    q.HitCount,
		},
	}, nil
}

func (b *bankServer) Save(ctx context.Context, req *hostv1.SaveRequest) (*hostv1.SaveResponse, error) {
	q := req.GetQuestion()
	if q == nil {
		return nil, status.Error(codes.InvalidArgument, "question is required")
	}
	_, err := b.svc.Save(ctx, &apiv1.QuestionSaveRequest{
		Platform:    q.GetPlatform(),
		QuestionKey: q.GetQuestionKey(),
		Stem:        q.GetStem(),
		Options:     q.GetOptions(),
		Answer:      q.GetAnswer(),
		Source:      q.GetSource(),
		Confidence:  q.GetConfidence(),
		Overwrite:   req.GetOverwrite(),
	})
	if err != nil {
		return nil, status.Error(codes.Internal, err.Error())
	}
	return &hostv1.SaveResponse{Saved: true}, nil
}
