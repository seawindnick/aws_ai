package repository

import (
	"context"
	"database/sql"
	"fmt"
	"time"

	"github.com/google/uuid"
	"github.com/workshop/wrong-question/internal/model"
)

type ErrorRecordRepo struct {
	db *sql.DB
}

func NewErrorRecordRepo(db *sql.DB) *ErrorRecordRepo {
	return &ErrorRecordRepo{db: db}
}

// Upsert increments wrong_count for (userID, questionID), inserting if not exists.
func (r *ErrorRecordRepo) Upsert(ctx context.Context, userID, questionID string) error {
	now := time.Now().UTC()
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO errors (id, user_id, question_id, wrong_count, last_wrong_at, created_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   wrong_count   = wrong_count + 1,
		   last_wrong_at = VALUES(last_wrong_at)`,
		uuid.New().String(), userID, questionID, now, now,
	)
	if err != nil {
		return fmt.Errorf("upsert error record: %w", err)
	}
	return nil
}

func (r *ErrorRecordRepo) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*model.ErrorRecord, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, question_id, wrong_count, last_wrong_at, created_at
		 FROM errors WHERE user_id = ? ORDER BY last_wrong_at DESC LIMIT ? OFFSET ?`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list error records: %w", err)
	}
	defer rows.Close()

	var result []*model.ErrorRecord
	for rows.Next() {
		e := &model.ErrorRecord{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.QuestionID, &e.WrongCount, &e.LastWrongAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan error record: %w", err)
		}
		result = append(result, e)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate error records: %w", err)
	}
	return result, nil
}

// SummarizeByUser returns error records for AI recommendation input.
func (r *ErrorRecordRepo) SummarizeByUser(ctx context.Context, userID string) ([]*model.ErrorRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, question_id, wrong_count, last_wrong_at, created_at
		 FROM errors WHERE user_id = ? ORDER BY wrong_count DESC, last_wrong_at DESC LIMIT 50`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("summarize error records: %w", err)
	}
	defer rows.Close()

	var result []*model.ErrorRecord
	for rows.Next() {
		e := &model.ErrorRecord{}
		if err := rows.Scan(&e.ID, &e.UserID, &e.QuestionID, &e.WrongCount, &e.LastWrongAt, &e.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan error record: %w", err)
		}
		result = append(result, e)
	}
	return result, rows.Err()
}
