package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type TagService struct {
	tagRepo      *repository.TagRepo
	questionRepo *repository.QuestionRepo
}

func NewTagService(tagRepo *repository.TagRepo, questionRepo *repository.QuestionRepo) *TagService {
	return &TagService{tagRepo: tagRepo, questionRepo: questionRepo}
}

// CreateSuggested 批量创建 suggested 标签，由识别结果触发。
func (s *TagService) CreateSuggested(ctx context.Context, questionID, userID string, names []string) error {
	for _, name := range names {
		if name == "" {
			continue
		}
		t := &model.Tag{
			ID:         uuid.New().String(),
			QuestionID: questionID,
			UserID:     userID,
			Name:       name,
			Status:     model.TagStatusSuggested,
			CreatedAt:  time.Now().UTC(),
		}
		if err := s.tagRepo.Create(ctx, t); err != nil {
			return fmt.Errorf("create suggested tag %q: %w", name, err)
		}
	}
	return nil
}

// AddManual 用户手动添加标签，直接为 confirmed 状态。
func (s *TagService) AddManual(ctx context.Context, questionID, userID, name string) (*model.Tag, error) {
	if name == "" {
		return nil, fmt.Errorf("tag name must not be empty")
	}
	if len(name) > 100 {
		return nil, fmt.Errorf("tag name too long (max 100 characters)")
	}
	// 验证题目归属（R4）
	if _, err := s.questionRepo.GetByID(ctx, questionID, userID); err != nil {
		return nil, fmt.Errorf("question not found or not owned by user: %w", err)
	}
	t := &model.Tag{
		ID:         uuid.New().String(),
		QuestionID: questionID,
		UserID:     userID,
		Name:       name,
		Status:     model.TagStatusConfirmed,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.tagRepo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("add manual tag: %w", err)
	}
	return t, nil
}

// List 返回题目的所有标签。
func (s *TagService) List(ctx context.Context, questionID, userID string) ([]*model.Tag, error) {
	// 验证题目归属（R4）
	if _, err := s.questionRepo.GetByID(ctx, questionID, userID); err != nil {
		return nil, fmt.Errorf("question not found or not owned by user: %w", err)
	}
	tags, err := s.tagRepo.ListByQuestion(ctx, questionID, userID)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	return tags, nil
}

// Confirm 接受建议标签，升级为 confirmed。
func (s *TagService) Confirm(ctx context.Context, tagID, userID string) error {
	if err := s.tagRepo.Confirm(ctx, tagID, userID); err != nil {
		return fmt.Errorf("confirm tag: %w", err)
	}
	return nil
}

// Delete 删除标签（不论状态）。
func (s *TagService) Delete(ctx context.Context, tagID, userID string) error {
	if err := s.tagRepo.Delete(ctx, tagID, userID); err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	return nil
}
