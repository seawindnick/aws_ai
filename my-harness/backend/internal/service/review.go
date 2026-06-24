package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type ReviewService struct {
	repo *repository.ReviewRepo
}

func NewReviewService(repo *repository.ReviewRepo) *ReviewService {
	return &ReviewService{repo: repo}
}

func (s *ReviewService) TodayList(ctx context.Context, userID string) ([]*model.ReviewSchedule, error) {
	schedules, err := s.repo.ListTodaySchedule(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get today schedule: %w", err)
	}
	return schedules, nil
}

func (s *ReviewService) SubmitResult(ctx context.Context, userID, questionID string, result model.ReviewResult) error {
	rec := &model.ReviewRecord{
		ID:         uuid.New().String(),
		UserID:     userID,
		QuestionID: questionID,
		ReviewedAt: time.Now().UTC(),
		Result:     string(result),
	}

	if err := s.repo.SaveRecord(ctx, rec); err != nil {
		return fmt.Errorf("save review record: %w", err)
	}

	var intervalDays int
	switch result {
	case model.ReviewPass:
		// TODO: 从 DynamoDB 查上次的 interval_days 再翻倍，首次默认为 1
		intervalDays = 2
	case model.ReviewFail:
		intervalDays = 1
	default:
		return fmt.Errorf("invalid review result: %s", result)
	}

	nextReview := time.Now().UTC().Add(time.Duration(intervalDays) * 24 * time.Hour)
	schedule := &model.ReviewSchedule{
		UserID:       userID,
		NextReviewAt: nextReview,
		QuestionID:   questionID,
		IntervalDays: intervalDays,
		TTL:          nextReview.Add(30 * 24 * time.Hour).Unix(),
	}

	if err := s.repo.SaveSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("save review schedule: %w", err)
	}

	return nil
}
