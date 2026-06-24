package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
)

type TaskRepo struct {
	db *sql.DB
}

func NewTaskRepo(db *sql.DB) *TaskRepo {
	return &TaskRepo{db: db}
}

func (r *TaskRepo) Create(ctx context.Context, t *model.ClassTask) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO class_tasks (id, class_id, paper_id, title, assigned_by, due_at, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		t.ID, t.ClassID, t.PaperID, t.Title, t.AssignedBy, t.DueAt, string(t.Status), t.CreatedAt,
	)
	return err
}

func (r *TaskRepo) GetByID(ctx context.Context, id string) (*model.ClassTask, error) {
	t := &model.ClassTask{}
	var dueAt sql.NullTime
	err := r.db.QueryRowContext(ctx,
		`SELECT id, class_id, paper_id, title, assigned_by, due_at, status, created_at
		 FROM class_tasks WHERE id = ?`, id,
	).Scan(&t.ID, &t.ClassID, &t.PaperID, &t.Title, &t.AssignedBy, &dueAt, &t.Status, &t.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get task: %w", err)
	}
	if dueAt.Valid {
		t.DueAt = &dueAt.Time
	}
	return t, nil
}

func (r *TaskRepo) ListByClass(ctx context.Context, classID string) ([]*model.ClassTask, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, class_id, paper_id, title, assigned_by, due_at, status, created_at
		 FROM class_tasks WHERE class_id = ? ORDER BY created_at DESC`,
		classID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanMany(rows)
}

func (r *TaskRepo) Update(ctx context.Context, id string, dueAt interface{}, status model.ClassTaskStatus) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE class_tasks SET due_at = ?, status = ? WHERE id = ?`,
		dueAt, string(status), id,
	)
	return err
}

func (r *TaskRepo) SaveSubmission(ctx context.Context, s *model.TaskSubmission) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO task_submissions (id, task_id, user_id, question_id, result, submitted_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		s.ID, s.TaskID, s.UserID, s.QuestionID, s.Result, s.SubmittedAt,
	)
	return err
}

func (r *TaskRepo) ListProgress(ctx context.Context, taskID string) ([]*model.TaskSubmission, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, task_id, user_id, question_id, result, submitted_at
		 FROM task_submissions WHERE task_id = ? ORDER BY submitted_at`,
		taskID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*model.TaskSubmission
	for rows.Next() {
		s := &model.TaskSubmission{}
		if err := rows.Scan(&s.ID, &s.TaskID, &s.UserID, &s.QuestionID, &s.Result, &s.SubmittedAt); err != nil {
			return nil, err
		}
		result = append(result, s)
	}
	return result, rows.Err()
}

func (r *TaskRepo) scanMany(rows *sql.Rows) ([]*model.ClassTask, error) {
	var result []*model.ClassTask
	for rows.Next() {
		t := &model.ClassTask{}
		var dueAt sql.NullTime
		if err := rows.Scan(&t.ID, &t.ClassID, &t.PaperID, &t.Title, &t.AssignedBy, &dueAt, &t.Status, &t.CreatedAt); err != nil {
			return nil, err
		}
		if dueAt.Valid {
			t.DueAt = &dueAt.Time
		}
		result = append(result, t)
	}
	return result, rows.Err()
}
