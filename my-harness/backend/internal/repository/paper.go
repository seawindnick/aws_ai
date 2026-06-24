package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
)

type PaperRepo struct {
	db *sql.DB
}

func NewPaperRepo(db *sql.DB) *PaperRepo {
	return &PaperRepo{db: db}
}

func (r *PaperRepo) Create(ctx context.Context, p *model.Paper) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO papers (id, user_id, title, status, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		p.ID, p.UserID, p.Title, p.Status, p.CreatedAt, p.UpdatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert paper: %w", err)
	}
	return nil
}

func (r *PaperRepo) ListByUser(ctx context.Context, userID string, page, pageSize int) ([]*model.Paper, error) {
	offset := (page - 1) * pageSize
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, title, status, created_at, updated_at
		 FROM papers WHERE user_id = ?
		 ORDER BY updated_at DESC LIMIT ? OFFSET ?`,
		userID, pageSize, offset,
	)
	if err != nil {
		return nil, fmt.Errorf("list papers: %w", err)
	}
	defer rows.Close()

	var result []*model.Paper
	for rows.Next() {
		p := &model.Paper{}
		if err := rows.Scan(&p.ID, &p.UserID, &p.Title, &p.Status, &p.CreatedAt, &p.UpdatedAt); err != nil {
			return nil, fmt.Errorf("scan paper: %w", err)
		}
		result = append(result, p)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate papers: %w", err)
	}
	return result, nil
}

func (r *PaperRepo) GetByID(ctx context.Context, id, userID string) (*model.Paper, error) {
	p := &model.Paper{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, user_id, title, status, created_at, updated_at
		 FROM papers WHERE id = ? AND user_id = ?`,
		id, userID,
	).Scan(&p.ID, &p.UserID, &p.Title, &p.Status, &p.CreatedAt, &p.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("get paper: %w", err)
	}
	return p, nil
}

func (r *PaperRepo) UpdateTitle(ctx context.Context, id, userID, title string) error {
	res, err := r.db.ExecContext(ctx,
		`UPDATE papers SET title = ? WHERE id = ? AND user_id = ?`,
		title, id, userID,
	)
	if err != nil {
		return fmt.Errorf("update paper title: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("paper not found or not owned by user")
	}
	return nil
}

func (r *PaperRepo) Delete(ctx context.Context, id, userID string) error {
	res, err := r.db.ExecContext(ctx,
		`DELETE FROM papers WHERE id = ? AND user_id = ?`, id, userID,
	)
	if err != nil {
		return fmt.Errorf("delete paper: %w", err)
	}
	affected, err := res.RowsAffected()
	if err != nil {
		return fmt.Errorf("rows affected: %w", err)
	}
	if affected == 0 {
		return fmt.Errorf("paper not found or not owned by user")
	}
	return nil
}

// AddQuestion 将题目加入试卷，position 由调用方指定。
func (r *PaperRepo) AddQuestion(ctx context.Context, pq *model.PaperQuestion) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO paper_questions (id, paper_id, question_id, position, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		pq.ID, pq.PaperID, pq.QuestionID, pq.Position, pq.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("add question to paper: %w", err)
	}
	return nil
}

// RemoveQuestion 从试卷移除题目。
func (r *PaperRepo) RemoveQuestion(ctx context.Context, paperID, questionID, userID string) error {
	// 先验证试卷归属（R4）
	var ownerID string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM papers WHERE id = ?`, paperID,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("verify paper ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("forbidden")
	}
	_, err = r.db.ExecContext(ctx,
		`DELETE FROM paper_questions WHERE paper_id = ? AND question_id = ?`,
		paperID, questionID,
	)
	if err != nil {
		return fmt.Errorf("remove question from paper: %w", err)
	}
	return nil
}

// ReorderQuestions 用一次事务整体替换试卷的题目顺序。
// positions: []struct{QuestionID, Position}
func (r *PaperRepo) ReorderQuestions(ctx context.Context, paperID, userID string, positions []struct {
	QuestionID string
	Position   int
}) error {
	var ownerID string
	err := r.db.QueryRowContext(ctx,
		`SELECT user_id FROM papers WHERE id = ?`, paperID,
	).Scan(&ownerID)
	if err != nil {
		return fmt.Errorf("verify paper ownership: %w", err)
	}
	if ownerID != userID {
		return fmt.Errorf("forbidden")
	}

	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin transaction: %w", err)
	}
	defer func() {
		if err != nil {
			_ = tx.Rollback()
		}
	}()

	for _, p := range positions {
		_, err = tx.ExecContext(ctx,
			`UPDATE paper_questions SET position = ? WHERE paper_id = ? AND question_id = ?`,
			p.Position, paperID, p.QuestionID,
		)
		if err != nil {
			return fmt.Errorf("update position for %s: %w", p.QuestionID, err)
		}
	}

	if err = tx.Commit(); err != nil {
		return fmt.Errorf("commit reorder: %w", err)
	}
	return nil
}

// ListQuestions 按 position 返回试卷内所有题目（含关联 question 详情）。
func (r *PaperRepo) ListQuestions(ctx context.Context, paperID, userID string) ([]*model.PaperQuestion, error) {
	// 验证归属
	var ownerID string
	err := r.db.QueryRowContext(ctx, `SELECT user_id FROM papers WHERE id = ?`, paperID).Scan(&ownerID)
	if err != nil {
		return nil, fmt.Errorf("get paper owner: %w", err)
	}
	if ownerID != userID {
		return nil, fmt.Errorf("forbidden")
	}

	rows, err := r.db.QueryContext(ctx,
		`SELECT pq.id, pq.paper_id, pq.question_id, pq.position, pq.created_at,
		        q.id, q.user_id, q.image_path, q.raw_text, q.subject, q.source,
		        q.status, q.category, q.confidence, q.review_note,
		        q.reviewed_by, q.reviewed_at, q.created_at
		 FROM paper_questions pq
		 JOIN questions q ON q.id = pq.question_id
		 WHERE pq.paper_id = ?
		 ORDER BY pq.position ASC`,
		paperID,
	)
	if err != nil {
		return nil, fmt.Errorf("list paper questions: %w", err)
	}
	defer rows.Close()

	var result []*model.PaperQuestion
	for rows.Next() {
		pq := &model.PaperQuestion{Question: &model.Question{}}
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		err := rows.Scan(
			&pq.ID, &pq.PaperID, &pq.QuestionID, &pq.Position, &pq.CreatedAt,
			&pq.Question.ID, &pq.Question.UserID, &pq.Question.ImagePath,
			&pq.Question.RawText, &pq.Question.Subject, &pq.Question.Source,
			&pq.Question.Status, &pq.Question.Category, &pq.Question.Confidence,
			&pq.Question.ReviewNote, &reviewedBy, &reviewedAt, &pq.Question.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan paper question: %w", err)
		}
		if reviewedBy.Valid {
			pq.Question.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			pq.Question.ReviewedAt = &t
		}
		result = append(result, pq)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate paper questions: %w", err)
	}
	return result, nil
}
