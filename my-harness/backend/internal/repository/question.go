package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"time"

	"github.com/workshop/wrong-question/internal/model"
)

type QuestionRepo struct {
	db *sql.DB
}

func NewQuestionRepo(db *sql.DB) *QuestionRepo {
	return &QuestionRepo{db: db}
}

func (r *QuestionRepo) Create(ctx context.Context, q *model.Question) error {
	tags, err := json.Marshal(q.TopicTags)
	if err != nil {
		return fmt.Errorf("marshal topic_tags: %w", err)
	}
	_, err = r.db.ExecContext(ctx,
		`INSERT INTO questions
		   (id, user_id, image_path, raw_text, subject, topic_tags, source,
		    status, category, confidence, review_note, reviewed_by, reviewed_at, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, '', NULL, NULL, ?)`,
		q.ID, q.UserID, q.ImagePath, q.RawText, q.Subject, string(tags), q.Source,
		q.Status, q.Category, q.Confidence, q.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert question: %w", err)
	}
	return nil
}

func (r *QuestionRepo) ListByUser(ctx context.Context, userID, subject string, status model.QuestionStatus, page, pageSize int) ([]*model.Question, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions
		 WHERE user_id = ?
		   AND (? = '' OR subject = ?)
		   AND (? = '' OR status = ?)
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, subject, subject, string(status), string(status), pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	return scanQuestions(rows)
}

// ListByStatus 不带 user_id 过滤，仅供教师/管理员审核使用。
func (r *QuestionRepo) ListByStatus(ctx context.Context, status model.QuestionStatus, page, pageSize int) ([]*model.Question, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions
		 WHERE status = ?
		 ORDER BY created_at ASC LIMIT ? OFFSET ?`,
		string(status), pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list questions by status: %w", err)
	}
	defer rows.Close()

	return scanQuestions(rows)
}

func (r *QuestionRepo) GetByID(ctx context.Context, id, userID string) (*model.Question, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions WHERE id = ? AND user_id = ?`,
		id, userID,
	)
	return scanQuestion(row)
}

// GetByIDNoUserFilter 供审核服务使用，不过滤 user_id（审核人员可跨用户查看）。
func (r *QuestionRepo) GetByIDNoUserFilter(ctx context.Context, id string) (*model.Question, error) {
	row := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions WHERE id = ?`,
		id,
	)
	return scanQuestion(row)
}

func (r *QuestionRepo) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM questions WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("question not found")
	}
	return nil
}

func (r *QuestionRepo) UpdateCategory(ctx context.Context, id, userID string, category model.QuestionCategory) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE questions SET category = ? WHERE id = ? AND user_id = ?`,
		string(category), id, userID,
	)
	if err != nil {
		return fmt.Errorf("update category: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("question not found or not owned by user")
	}
	return nil
}

func (r *QuestionRepo) UpdateReviewResult(
	ctx context.Context,
	id string,
	status model.QuestionStatus,
	category model.QuestionCategory,
	note, reviewerID string,
	reviewedAt time.Time,
) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE questions
		 SET status = ?, category = ?, review_note = ?, reviewed_by = ?, reviewed_at = ?
		 WHERE id = ?`,
		string(status), string(category), note, reviewerID, reviewedAt, id,
	)
	if err != nil {
		return fmt.Errorf("update review result: %w", err)
	}
	return nil
}

func scanQuestions(rows *sql.Rows) ([]*model.Question, error) {
	var result []*model.Question
	for rows.Next() {
		q, err := scanQuestionRow(rows)
		if err != nil {
			return nil, err
		}
		result = append(result, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate questions: %w", err)
	}
	return result, nil
}

type scanner interface {
	Scan(dest ...any) error
}

func scanQuestion(row *sql.Row) (*model.Question, error) {
	return scanQuestionRow(row)
}

func scanQuestionRow(s scanner) (*model.Question, error) {
	q := &model.Question{}
	var tagsJSON string
	var reviewedBy sql.NullString
	var reviewedAt sql.NullTime

	err := s.Scan(
		&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &tagsJSON, &q.Source,
		&q.Status, &q.Category, &q.Confidence, &q.ReviewNote, &reviewedBy, &reviewedAt,
		&q.CreatedAt,
	)
	if err != nil {
		return nil, fmt.Errorf("scan question: %w", err)
	}

	if err := json.Unmarshal([]byte(tagsJSON), &q.TopicTags); err != nil {
		return nil, fmt.Errorf("unmarshal topic_tags: %w", err)
	}
	if reviewedBy.Valid {
		q.ReviewedBy = reviewedBy.String
	}
	if reviewedAt.Valid {
		t := reviewedAt.Time
		q.ReviewedAt = &t
	}
	return q, nil
}
