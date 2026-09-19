package service

import (
	"context"

	v1 "backend/api/v1"
	"backend/internal/model"
	"backend/internal/repository"
	"backend/pkg/log"
	"backend/pkg/sid"
)

type QuestionService struct {
	repo   *repository.QuestionRepository
	sid    *sid.Sid
	logger *log.Logger
}

func NewQuestionService(repo *repository.QuestionRepository, sid *sid.Sid, logger *log.Logger) *QuestionService {
	return &QuestionService{repo: repo, sid: sid, logger: logger}
}

func (s *QuestionService) Lookup(ctx context.Context, platform, questionKey string) (*v1.QuestionResponse, error) {
	if !IsPlatformSupported(platform) {
		return nil, v1.ErrUnsupportedPlatform
	}
	q, err := s.repo.Lookup(ctx, platform, questionKey)
	if err != nil {
		return nil, err
	}
	return toQuestionResponse(q), nil
}

func (s *QuestionService) Save(ctx context.Context, req *v1.QuestionSaveRequest) (*v1.QuestionResponse, error) {
	if !IsPlatformSupported(req.Platform) {
		return nil, v1.ErrUnsupportedPlatform
	}
	source := req.Source
	if source == "" {
		source = "bank"
	}
	options := req.Options
	if options == "" {
		options = "[]"
	}
	answer := req.Answer
	if answer == "" {
		answer = "{}"
	}
	id, err := s.sid.GenString()
	if err != nil {
		return nil, err
	}
	q := &model.Question{
		ID:          id,
		Platform:    req.Platform,
		QuestionKey: req.QuestionKey,
		Stem:        req.Stem,
		Options:     options,
		Answer:      answer,
		Source:      source,
		Confidence:  req.Confidence,
	}
	if _, err := s.repo.Save(ctx, q, req.Overwrite); err != nil {
		return nil, err
	}
	saved, err := s.repo.Lookup(ctx, req.Platform, req.QuestionKey)
	if err != nil {
		return nil, err
	}
	return toQuestionResponse(saved), nil
}

func (s *QuestionService) List(ctx context.Context, platform string, page, size int) ([]v1.QuestionResponse, int64, error) {
	if page < 1 {
		page = 1
	}
	if size < 1 || size > 100 {
		size = 20
	}
	questions, total, err := s.repo.List(ctx, platform, size, (page-1)*size)
	if err != nil {
		return nil, 0, err
	}
	out := make([]v1.QuestionResponse, 0, len(questions))
	for i := range questions {
		out = append(out, *toQuestionResponse(&questions[i]))
	}
	return out, total, nil
}

func toQuestionResponse(q *model.Question) *v1.QuestionResponse {
	return &v1.QuestionResponse{
		ID:          q.ID,
		Platform:    q.Platform,
		QuestionKey: q.QuestionKey,
		Stem:        q.Stem,
		Options:     q.Options,
		Answer:      q.Answer,
		Source:      q.Source,
		Confidence:  q.Confidence,
		HitCount:    q.HitCount,
		CreatedAt:   q.CreatedAt,
		UpdatedAt:   q.UpdatedAt,
	}
}
