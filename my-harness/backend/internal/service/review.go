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
	repo      *repository.ReviewRepo
	errorRepo *repository.ErrorRecordRepo
}

func NewReviewService(repo *repository.ReviewRepo, errorRepo *repository.ErrorRecordRepo) *ReviewService {
	return &ReviewService{repo: repo, errorRepo: errorRepo}
}

func (s *ReviewService) TodayList(ctx context.Context, userID string) ([]*model.ReviewSchedule, error) {
	schedules, err := s.repo.ListTodaySchedule(ctx, userID)
	if err != nil {
		return nil, fmt.Errorf("get today schedule: %w", err)
	}
	return schedules, nil
}

func (s *ReviewService) SubmitResult(ctx context.Context, userID, questionID string, result model.ReviewResult) error {
	if result != model.ReviewPass && result != model.ReviewFail {
		return fmt.Errorf("invalid review result: %s", result)
	}

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

	existing, err := s.repo.GetSchedule(ctx, userID, questionID)
	if err != nil {
		return fmt.Errorf("get schedule: %w", err)
	}

	var prevInterval int
	if existing != nil {
		prevInterval = existing.IntervalDays
	}

	var newInterval int
	switch result {
	case model.ReviewPass:
		if prevInterval < 1 {
			newInterval = 1
		} else {
			newInterval = prevInterval * 2
		}
	case model.ReviewFail:
		newInterval = 1
		if err := s.errorRepo.Upsert(ctx, userID, questionID); err != nil {
			return fmt.Errorf("upsert error record: %w", err)
		}
	}

	nextReview := time.Now().UTC().Add(time.Duration(newInterval) * 24 * time.Hour)
	schedule := &model.ReviewSchedule{
		UserID:       userID,
		QuestionID:   questionID,
		NextReviewAt: nextReview,
		IntervalDays: newInterval,
		TTL:          nextReview.Add(30 * 24 * time.Hour).Unix(),
	}
	if err := s.repo.SaveSchedule(ctx, schedule); err != nil {
		return fmt.Errorf("save review schedule: %w", err)
	}

	return nil
}
