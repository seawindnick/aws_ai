package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
)

type ErrorRecordRepo struct {
	db *sql.DB
}

func NewErrorRecordRepo(db *sql.DB) *ErrorRecordRepo {
	return &ErrorRecordRepo{db: db}
}

func (r *ErrorRecordRepo) Upsert(ctx context.Context, e *model.ErrorRecord) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO errors (id, user_id, question_id, wrong_count, last_wrong_at, created_at)
		 VALUES (?, ?, ?, 1, ?, ?)
		 ON DUPLICATE KEY UPDATE
		   wrong_count = wrong_count + 1,
		   last_wrong_at = VALUES(last_wrong_at)`,
		e.ID, e.UserID, e.QuestionID, e.LastWrongAt, e.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("upsert error record: %w", err)
	}
	return nil
}

func (r *ErrorRecordRepo) ListByUser(ctx context.Context, userID string) ([]*model.ErrorRecord, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, question_id, wrong_count, last_wrong_at, created_at
		 FROM errors WHERE user_id = ? ORDER BY last_wrong_at DESC`,
		userID,
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
