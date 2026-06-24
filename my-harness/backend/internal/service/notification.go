package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type NotificationService struct {
	repo *repository.NotificationRepo
}

func NewNotificationService(repo *repository.NotificationRepo) *NotificationService {
	return &NotificationService{repo: repo}
}

// NotifyQuestionReviewed 审核完成后通知题目所有者（REQ-REVIEW-06）。
func (s *NotificationService) NotifyQuestionReviewed(ctx context.Context, userID, questionID string, status model.QuestionStatus) error {
	var title, body string
	var notifType model.NotificationType

	switch status {
	case model.QuestionStatusApproved:
		notifType = model.NotifTypeQuestionApproved
		title = "题目审核通过"
		body = fmt.Sprintf("您上传的题目（ID: %s）已通过审核，已加入您的题库。", questionID)
	case model.QuestionStatusRejected:
		notifType = model.NotifTypeQuestionRejected
		title = "题目审核未通过"
		body = fmt.Sprintf("您上传的题目（ID: %s）未通过审核，请查看审核备注后重新上传。", questionID)
	default:
		return fmt.Errorf("unsupported review status for notification: %s", status)
	}

	n := &model.Notification{
		ID:        uuid.New().String(),
		UserID:    userID,
		Type:      notifType,
		Title:     title,
		Body:      body,
		RefID:     questionID,
		CreatedAt: time.Now().UTC(),
	}
	if err := s.repo.Create(ctx, n); err != nil {
		return fmt.Errorf("create review notification: %w", err)
	}
	return nil
}

func (s *NotificationService) List(ctx context.Context, userID string, onlyUnread bool, page, pageSize int) ([]*model.Notification, error) {
	items, err := s.repo.ListByUser(ctx, userID, onlyUnread, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list notifications: %w", err)
	}
	return items, nil
}

func (s *NotificationService) MarkRead(ctx context.Context, id, userID string) error {
	if err := s.repo.MarkRead(ctx, id, userID); err != nil {
		return fmt.Errorf("mark notification read: %w", err)
	}
	return nil
}

func (s *NotificationService) MarkAllRead(ctx context.Context, userID string) error {
	if err := s.repo.MarkAllRead(ctx, userID); err != nil {
		return fmt.Errorf("mark all notifications read: %w", err)
	}
	return nil
}
