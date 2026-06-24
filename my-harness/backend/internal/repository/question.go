package repository

import (
	"context"
	"fmt"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/workshop/wrong-question/internal/model"
)

type QuestionRepo struct {
	db *pgxpool.Pool
}

func NewQuestionRepo(db *pgxpool.Pool) *QuestionRepo {
	return &QuestionRepo{db: db}
}

func (r *QuestionRepo) Create(ctx context.Context, q *model.Question) error {
	_, err := r.db.Exec(ctx,
		`INSERT INTO questions (id, user_id, image_path, raw_text, subject, topic_tags, source, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7, $8)`,
		q.ID, q.UserID, q.ImagePath, q.RawText, q.Subject, q.TopicTags, q.Source, q.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert question: %w", err)
	}
	return nil
}

func (r *QuestionRepo) ListByUser(ctx context.Context, userID, subject string, page, pageSize int) ([]*model.Question, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.Query(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source, created_at
		 FROM questions
		 WHERE user_id = $1 AND ($2 = '' OR subject = $2)
		 ORDER BY created_at DESC LIMIT $3 OFFSET $4`,
		userID, subject, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list questions: %w", err)
	}
	defer rows.Close()

	var result []*model.Question
	for rows.Next() {
		q := &model.Question{}
		if err := rows.Scan(&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &q.TopicTags, &q.Source, &q.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan question: %w", err)
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
	err := r.db.QueryRow(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, topic_tags, source, created_at
		 FROM questions WHERE id = $1 AND user_id = $2`,
		id, userID,
	).Scan(&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &q.TopicTags, &q.Source, &q.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get question: %w", err)
	}
	return q, nil
}

func (r *QuestionRepo) Delete(ctx context.Context, id, userID string) error {
	tag, err := r.db.Exec(ctx,
		`DELETE FROM questions WHERE id = $1 AND user_id = $2`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete question: %w", err)
	}
	if tag.RowsAffected() == 0 {
		return fmt.Errorf("question not found")
	}
	return nil
}
