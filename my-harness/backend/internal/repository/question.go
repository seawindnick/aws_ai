package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"

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
		`INSERT INTO questions (id, user_id, image_path, raw_text, subject, topic_tags, source, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		q.ID, q.UserID, q.ImagePath, q.RawText, q.Subject, string(tags), q.Source, q.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert question: %w", err)
	}
	return nil
}

func (r *QuestionRepo) ListByUser(ctx context.Context, userID, subject string, page, pageSize int) ([]*model.Question, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source, created_at
		 FROM questions
		 WHERE user_id = ? AND (? = '' OR subject = ?)
		 ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		userID, subject, subject, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	var result []*model.Question
	for rows.Next() {
		q := &model.Question{}
		var tagsJSON string
		if err := rows.Scan(&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &tagsJSON, &q.Source, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
		}
		if err := json.Unmarshal([]byte(tagsJSON), &q.TopicTags); err != nil {
			return nil, fmt.Errorf("unmarshal topic_tags: %w", err)
		}
		result = append(result, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate questions: %w", err)
	}
	return result, nil
}

func (r *QuestionRepo) GetByID(ctx context.Context, id, userID string) (*model.Question, error) {
	q := &model.Question{}
	var tagsJSON string
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source, created_at
		 FROM questions WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &tagsJSON, &q.Source, &q.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get question: %w", err)
	}
	if err := json.Unmarshal([]byte(tagsJSON), &q.TopicTags); err != nil {
		return nil, fmt.Errorf("unmarshal topic_tags: %w", err)
	}
	return q, nil
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
