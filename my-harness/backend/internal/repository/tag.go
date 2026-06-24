package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
)

type TagRepo struct {
	db *sql.DB
}

func NewTagRepo(db *sql.DB) *TagRepo {
	return &TagRepo{db: db}
}

func (r *TagRepo) Create(ctx context.Context, t *model.Tag) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO question_tags (id, question_id, user_id, name, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		t.ID, t.QuestionID, t.UserID, t.Name, t.Status, t.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert tag: %w", err)
	}
	return nil
}

func (r *TagRepo) ListByQuestion(ctx context.Context, questionID, userID string) ([]*model.Tag, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, question_id, user_id, name, status, created_at
		 FROM question_tags
		 WHERE question_id = ? AND user_id = ?
		 ORDER BY status DESC, created_at ASC`,
		questionID, userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list tags: %w", err)
	}
	defer rows.Close()

	var result []*model.Tag
	for rows.Next() {
		t := &model.Tag{}
		if err := rows.Scan(&t.ID, &t.QuestionID, &t.UserID, &t.Name, &t.Status, &t.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan tag: %w", err)
		}
		result = append(result, t)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate tags: %w", err)
	}
	return result, nil
}

// Confirm 将 suggested 标签升为 confirmed。
func (r *TagRepo) Confirm(ctx context.Context, tagID, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE question_tags SET status = 'confirmed'
		 WHERE id = ? AND user_id = ? AND status = 'suggested'`,
		tagID, userID,
	)
	if err != nil {
		return fmt.Errorf("confirm tag: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("tag not found, not owned by user, or already confirmed")
	}
	return nil
}

// DeleteByQuestion removes all tags for a question (used on question delete cascade).
func (r *TagRepo) DeleteByQuestion(ctx context.Context, questionID string) error {
	_, err := r.db.ExecContext(ctx, `DELETE FROM question_tags WHERE question_id = ?`, questionID)
	if err != nil {
		return fmt.Errorf("delete tags by question: %w", err)
	}
	return nil
}

// Delete 删除用户自己的标签（无论状态）。
func (r *TagRepo) Delete(ctx context.Context, tagID, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM question_tags WHERE id = ? AND user_id = ?`,
		tagID, userID,
	)
	if err != nil {
		return fmt.Errorf("delete tag: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("tag not found or not owned by user")
	}
	return nil
}
