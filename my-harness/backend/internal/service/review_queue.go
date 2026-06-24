package service

import (
	"context"
	"fmt"
	"time"

	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

// ReviewQueueService 处理人工审核队列，仅教师/管理员可调用。
type ReviewQueueService struct {
	questionRepo *repository.QuestionRepo
}

func NewReviewQueueService(questionRepo *repository.QuestionRepo) *ReviewQueueService {
	return &ReviewQueueService{questionRepo: questionRepo}
}

// ListPending 列出所有待审核题目，供审核人员查阅。
func (s *ReviewQueueService) ListPending(ctx context.Context, page, pageSize int) ([]*model.Question, error) {
	questions, err := s.questionRepo.ListByStatus(ctx, model.QuestionStatusPendingReview, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list pending review questions: %w", err)
	}
	return questions, nil
}

// Review 对单道题目执行审核操作（通过或拒绝），同时支持修正分类和备注。
// reviewerID 必须来自鉴权 context，不得由客户端传入（R4）。
func (s *ReviewQueueService) Review(ctx context.Context, questionID, reviewerID string, action model.ReviewAction, category model.QuestionCategory, note string) error {
	q, err := s.questionRepo.GetByIDNoUserFilter(ctx, questionID)
	if err != nil {
		return fmt.Errorf("get question for review: %w", err)
	}

	if q.Status != model.QuestionStatusPendingReview {
		return fmt.Errorf("question %s is not pending review (current status: %s)", questionID, q.Status)
	}

	var newStatus model.QuestionStatus
	switch action {
	case model.ReviewActionApprove:
		newStatus = model.QuestionStatusApproved
	case model.ReviewActionReject:
		newStatus = model.QuestionStatusRejected
	default:
		return fmt.Errorf("invalid review action: %s", action)
	}

	if category != "" && !isValidCategory(category) {
		return fmt.Errorf("invalid category: %s", category)
	}
	finalCategory := q.Category
	if category != "" {
		finalCategory = category
	}

	now := time.Now().UTC()
	if err := s.questionRepo.UpdateReviewResult(ctx, questionID, newStatus, finalCategory, note, reviewerID, now); err != nil {
		return fmt.Errorf("save review result: %w", err)
	}

	return nil
}
