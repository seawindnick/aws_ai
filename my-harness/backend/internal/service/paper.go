package service

import (
	"context"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type PaperService struct {
	paperRepo    *repository.PaperRepo
	questionRepo *repository.QuestionRepo
}

func NewPaperService(paperRepo *repository.PaperRepo, questionRepo *repository.QuestionRepo) *PaperService {
	return &PaperService{paperRepo: paperRepo, questionRepo: questionRepo}
}

// Create 新建草稿试卷（REQ-PAPER-01）。
func (s *PaperService) Create(ctx context.Context, userID, title string) (*model.Paper, error) {
	if title == "" {
		return nil, fmt.Errorf("paper title must not be empty")
	}
	now := time.Now().UTC()
	p := &model.Paper{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     title,
		Status:    model.PaperStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.paperRepo.Create(ctx, p); err != nil {
		return nil, fmt.Errorf("create paper: %w", err)
	}
	return p, nil
}

func (s *PaperService) List(ctx context.Context, userID string, page, pageSize int) ([]*model.Paper, error) {
	papers, err := s.paperRepo.ListByUser(ctx, userID, page, pageSize)
	if err != nil {
		return nil, fmt.Errorf("list papers: %w", err)
	}
	return papers, nil
}

func (s *PaperService) Get(ctx context.Context, id, userID string) (*model.Paper, error) {
	p, err := s.paperRepo.GetByID(ctx, id, userID)
	if err != nil {
		return nil, fmt.Errorf("get paper: %w", err)
	}
	return p, nil
}

func (s *PaperService) Rename(ctx context.Context, id, userID, title string) error {
	if title == "" {
		return fmt.Errorf("paper title must not be empty")
	}
	if err := s.paperRepo.UpdateTitle(ctx, id, userID, title); err != nil {
		return fmt.Errorf("rename paper: %w", err)
	}
	return nil
}

func (s *PaperService) Delete(ctx context.Context, id, userID string) error {
	if err := s.paperRepo.Delete(ctx, id, userID); err != nil {
		return fmt.Errorf("delete paper: %w", err)
	}
	return nil
}

// AddQuestion 将已 approved 的题目加入试卷（REQ-PAPER-07 防御）。
func (s *PaperService) AddQuestion(ctx context.Context, paperID, questionID, userID string, position int) error {
	// 验证试卷归属
	if _, err := s.paperRepo.GetByID(ctx, paperID, userID); err != nil {
		return fmt.Errorf("paper not found or not owned by user: %w", err)
	}
	// 验证题目归属并检查状态
	q, err := s.questionRepo.GetByID(ctx, questionID, userID)
	if err != nil {
		return fmt.Errorf("question not found or not owned by user: %w", err)
	}
	if q.Status != model.QuestionStatusApproved {
		return fmt.Errorf("only approved questions can be added to a paper (current status: %s)", q.Status)
	}

	pq := &model.PaperQuestion{
		ID:         uuid.New().String(),
		PaperID:    paperID,
		QuestionID: questionID,
		Position:   position,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.paperRepo.AddQuestion(ctx, pq); err != nil {
		return fmt.Errorf("add question to paper: %w", err)
	}
	return nil
}

// RemoveQuestion 从试卷移除题目。
func (s *PaperService) RemoveQuestion(ctx context.Context, paperID, questionID, userID string) error {
	if err := s.paperRepo.RemoveQuestion(ctx, paperID, questionID, userID); err != nil {
		return fmt.Errorf("remove question from paper: %w", err)
	}
	return nil
}

// Reorder 整体更新试卷题目顺序。
func (s *PaperService) Reorder(ctx context.Context, paperID, userID string, positions []struct {
	QuestionID string
	Position   int
}) error {
	if len(positions) == 0 {
		return fmt.Errorf("positions must not be empty")
	}
	if err := s.paperRepo.ReorderQuestions(ctx, paperID, userID, positions); err != nil {
		return fmt.Errorf("reorder paper questions: %w", err)
	}
	return nil
}

// ListQuestions 列出试卷题目（按 position 排序）。
func (s *PaperService) ListQuestions(ctx context.Context, paperID, userID string) ([]*model.PaperQuestion, error) {
	items, err := s.paperRepo.ListQuestions(ctx, paperID, userID)
	if err != nil {
		return nil, fmt.Errorf("list paper questions: %w", err)
	}
	return items, nil
}

// Duplicate 复制试卷为新草稿（REQ-PAPER-09）。
func (s *PaperService) Duplicate(ctx context.Context, paperID, userID string) (*model.Paper, error) {
	src, err := s.paperRepo.GetByID(ctx, paperID, userID)
	if err != nil {
		return nil, fmt.Errorf("source paper not found or not owned by user: %w", err)
	}

	now := time.Now().UTC()
	dst := &model.Paper{
		ID:        uuid.New().String(),
		UserID:    userID,
		Title:     src.Title + " (副本)",
		Status:    model.PaperStatusDraft,
		CreatedAt: now,
		UpdatedAt: now,
	}
	if err := s.paperRepo.Create(ctx, dst); err != nil {
		return nil, fmt.Errorf("create duplicate paper: %w", err)
	}

	srcItems, err := s.paperRepo.ListQuestions(ctx, paperID, userID)
	if err != nil {
		return nil, fmt.Errorf("list source paper questions: %w", err)
	}
	for _, item := range srcItems {
		pq := &model.PaperQuestion{
			ID:         uuid.New().String(),
			PaperID:    dst.ID,
			QuestionID: item.QuestionID,
			Position:   item.Position,
			CreatedAt:  now,
		}
		if err := s.paperRepo.AddQuestion(ctx, pq); err != nil {
			return nil, fmt.Errorf("copy question %s to duplicate: %w", item.QuestionID, err)
		}
	}
	return dst, nil
}
