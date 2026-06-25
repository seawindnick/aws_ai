package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type ReviewQueueService struct {
	questionRepo *repository.QuestionRepo
	notifSvc     *NotificationService
}

func NewReviewQueueService(questionRepo *repository.QuestionRepo, notifSvc *NotificationService) *ReviewQueueService {
	return &ReviewQueueService{questionRepo: questionRepo, notifSvc: notifSvc}
}

func (s *ReviewQueueService) ListPending(ctx context.Context, page, pageSize int) ([]*model.Question, error) {
	questions, err := s.questionRepo.ListByStatus(ctx, model.QuestionStatusPendingReview, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list pending review questions: %w", err)
	}
	return questions, nil
}

// Review 审核题目，完成后发送站内通知（REQ-REVIEW-06）。
// reviewerID 来自鉴权 context，不得由客户端传入（R4）。
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

	// 通知题目所有者（REQ-REVIEW-06）；通知失败不回滚审核结果（弱依赖）
	if err := s.notifSvc.NotifyQuestionReviewed(ctx, q.UserID, questionID, newStatus); err != nil {
		slog.Warn("send review notification failed", "question_id", questionID, "error", err)
	}

	return nil
}
