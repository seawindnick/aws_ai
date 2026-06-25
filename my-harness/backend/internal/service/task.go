package service

import (
	"context"
	"fmt"
	"log/slog"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
	"github.com/workshop/wrong-question/internal/repository"
)

type TaskService struct {
	taskRepo  *repository.TaskRepo
	classRepo *repository.ClassRepo
	reviewSvc *ReviewService
}

func NewTaskService(taskRepo *repository.TaskRepo, classRepo *repository.ClassRepo, reviewSvc *ReviewService) *TaskService {
	return &TaskService{taskRepo: taskRepo, classRepo: classRepo, reviewSvc: reviewSvc}
}

func (s *TaskService) Create(ctx context.Context, teacherID, classID, paperID, title string, dueAt *time.Time) (*model.ClassTask, error) {
	c, err := s.classRepo.GetByID(ctx, classID)
	if err != nil {
		return nil, fmt.Errorf("class not found")
	}
	if c.TeacherID != teacherID {
		return nil, fmt.Errorf("forbidden")
	}
	t := &model.ClassTask{
		ID:         uuid.New().String(),
		ClassID:    classID,
		PaperID:    paperID,
		Title:      title,
		AssignedBy: teacherID,
		DueAt:      dueAt,
		Status:     model.ClassTaskStatusActive,
		CreatedAt:  time.Now().UTC(),
	}
	if err := s.taskRepo.Create(ctx, t); err != nil {
		return nil, fmt.Errorf("create task: %w", err)
	}
	return t, nil
}

func (s *TaskService) List(ctx context.Context, classID string) ([]*model.ClassTask, error) {
	return s.taskRepo.ListByClass(ctx, classID)
}

func (s *TaskService) Get(ctx context.Context, taskID string) (*model.ClassTask, error) {
	return s.taskRepo.GetByID(ctx, taskID)
}

func (s *TaskService) Update(ctx context.Context, taskID, teacherID string, dueAt *time.Time, status model.ClassTaskStatus) error {
	t, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return fmt.Errorf("task not found")
	}
	if t.AssignedBy != teacherID {
		return fmt.Errorf("forbidden")
	}
	if t.Status == model.ClassTaskStatusClosed {
		return fmt.Errorf("closed task cannot be updated")
	}
	return s.taskRepo.Update(ctx, taskID, dueAt, status)
}

type SubmitResult struct {
	Succeeded []string `json:"succeeded"`
	Failed    []struct {
		QuestionID string `json:"question_id"`
		Reason     string `json:"reason"`
	} `json:"failed"`
}

func (s *TaskService) Submit(ctx context.Context, userID, taskID string, results []struct {
	QuestionID string
	Result     model.ReviewResult
}) (*SubmitResult, error) {
	t, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found")
	}
	if t.Status == model.ClassTaskStatusClosed {
		return nil, fmt.Errorf("task is closed")
	}
	if t.DueAt != nil && time.Now().UTC().After(*t.DueAt) {
		return nil, fmt.Errorf("task due date has passed")
	}

	isMember, err := s.classRepo.IsMember(ctx, t.ClassID, userID)
	if err != nil || !isMember {
		return nil, fmt.Errorf("not a member of this class")
	}

	out := &SubmitResult{}
	for _, item := range results {
		sub := &model.TaskSubmission{
			ID:          uuid.New().String(),
			TaskID:      taskID,
			UserID:      userID,
			QuestionID:  item.QuestionID,
			Result:      string(item.Result),
			SubmittedAt: time.Now().UTC(),
		}
		if err := s.taskRepo.SaveSubmission(ctx, sub); err != nil {
			out.Failed = append(out.Failed, struct {
				QuestionID string `json:"question_id"`
				Reason     string `json:"reason"`
			}{QuestionID: item.QuestionID, Reason: "duplicate submission"})
			continue
		}
		// trigger Ebbinghaus schedule update
		if err := s.reviewSvc.SubmitResult(ctx, userID, item.QuestionID, item.Result); err != nil {
			slog.Warn("review schedule update failed", "question_id", item.QuestionID, "error", err)
		}
		out.Succeeded = append(out.Succeeded, item.QuestionID)
	}
	return out, nil
}

func (s *TaskService) Progress(ctx context.Context, taskID, teacherID string) ([]*model.TaskSubmission, error) {
	t, err := s.taskRepo.GetByID(ctx, taskID)
	if err != nil {
		return nil, fmt.Errorf("task not found")
	}
	c, err := s.classRepo.GetByID(ctx, t.ClassID)
	if err != nil {
		return nil, fmt.Errorf("class not found")
	}
	if c.TeacherID != teacherID {
		return nil, fmt.Errorf("forbidden")
	}
	return s.taskRepo.ListProgress(ctx, taskID)
}
