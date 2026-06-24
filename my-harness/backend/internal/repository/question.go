package repository

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
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

// Search 多维搜索，仅返回 userID 归属的题目（R4）。
func (r *QuestionRepo) Search(ctx context.Context, p model.SearchParams) ([]*model.Question, error) {
	offset := (p.Page - 1) * p.PageSize
	where := []string{"q.user_id = ?"}
	args := []any{p.UserID}

	if p.Subject != "" {
		where = append(where, "q.subject = ?")
		args = append(args, p.Subject)
	}
	if p.Category != "" {
		where = append(where, "q.category = ?")
		args = append(args, string(p.Category))
	}
	if p.Status != "" {
		where = append(where, "q.status = ?")
		args = append(args, string(p.Status))
	}
	if p.Keyword != "" {
		where = append(where, "q.raw_text LIKE ?")
		args = append(args, "%"+p.Keyword+"%")
	}
	if p.DateFrom != nil {
		where = append(where, "q.created_at >= ?")
		args = append(args, *p.DateFrom)
	}
	if p.DateTo != nil {
		where = append(where, "q.created_at <= ?")
		args = append(args, *p.DateTo)
	}
	if p.Tag != "" {
		where = append(where, `EXISTS (
			SELECT 1 FROM question_tags qt
			WHERE qt.question_id = q.id AND qt.name = ? AND qt.status = 'confirmed'
		)`)
		args = append(args, p.Tag)
	}

	query := `SELECT q.id, q.user_id, q.image_path, q.raw_text, q.subject, q.source,
	                 q.status, q.category, q.confidence, q.review_note,
	                 q.reviewed_by, q.reviewed_at, q.created_at
	          FROM questions q
	          WHERE ` + strings.Join(where, " AND ") +
		` ORDER BY q.created_at DESC LIMIT ? OFFSET ?`
	args = append(args, p.PageSize, offset)

	rows, err := r.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, fmt.Errorf("search questions: %w", err)
	}
	defer rows.Close()

	return scanQuestionsNoTags(rows)
}

func (r *QuestionRepo) ListApprovedByUser(ctx context.Context, userID string) ([]*model.Question, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions WHERE user_id = ? AND status = 'approved'`,
		userID,
	)
	if err != nil {
		return nil, fmt.Errorf("list approved questions: %w", err)
	}
	defer rows.Close()
	return scanQuestionsNoTags(rows)
}

func (r *QuestionRepo) DailyNewCounts(ctx context.Context, userID string, from, to time.Time) ([]struct {
	Date  string
	Count int
}, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT DATE(created_at) as d, COUNT(*) as cnt
		 FROM questions
		 WHERE user_id = ? AND created_at >= ? AND created_at <= ?
		 GROUP BY d ORDER BY d`,
		userID, from, to,
	)
	if err != nil {
		return nil, fmt.Errorf("daily new counts: %w", err)
	}
	defer rows.Close()
	var result []struct {
		Date  string
		Count int
	}
	for rows.Next() {
		var d string
		var cnt int
		if err := rows.Scan(&d, &cnt); err != nil {
			return nil, err
		}
		result = append(result, struct {
			Date  string
			Count int
		}{Date: d, Count: cnt})
	}
	return result, rows.Err()
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

func (r *QuestionRepo) DeleteCascadeRelated(ctx context.Context, questionID string) error {
	for _, stmt := range []string{
		`DELETE FROM paper_questions WHERE question_id = ?`,
		`DELETE FROM errors WHERE question_id = ?`,
		`DELETE FROM review_records WHERE question_id = ?`,
	} {
		if _, err := r.db.ExecContext(ctx, stmt, questionID); err != nil {
			return fmt.Errorf("cascade delete (%s): %w", stmt, err)
		}
	}
	return nil
}

func (r *QuestionRepo) GetByIDs(ctx context.Context, userID string, ids []string) ([]*model.Question, error) {
	if len(ids) == 0 {
		return nil, nil
	}
	placeholders := strings.Repeat("?,", len(ids))
	placeholders = placeholders[:len(placeholders)-1]
	args := []any{userID}
	for _, id := range ids {
		args = append(args, id)
	}
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, user_id, image_path, raw_text, subject, source,
		        status, category, confidence, review_note, reviewed_by, reviewed_at, created_at
		 FROM questions WHERE user_id = ? AND id IN (`+placeholders+`)`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("get questions by ids: %w", err)
	}
	defer rows.Close()
	return scanQuestionsNoTags(rows)
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

// scanQuestionsNoTags 用于搜索结果，不含 topic_tags 列（由 tag repo 单独加载）。
func scanQuestionsNoTags(rows *sql.Rows) ([]*model.Question, error) {
	var result []*model.Question
	for rows.Next() {
		q := &model.Question{}
		var reviewedBy sql.NullString
		var reviewedAt sql.NullTime
		err := rows.Scan(
			&q.ID, &q.UserID, &q.ImagePath, &q.RawText, &q.Subject, &q.Source,
			&q.Status, &q.Category, &q.Confidence, &q.ReviewNote,
			&reviewedBy, &reviewedAt, &q.CreatedAt,
		)
		if err != nil {
			return nil, fmt.Errorf("scan question (no tags): %w", err)
		}
		if reviewedBy.Valid {
			q.ReviewedBy = reviewedBy.String
		}
		if reviewedAt.Valid {
			t := reviewedAt.Time
			q.ReviewedAt = &t
		}
		result = append(result, q)
	}
	if err := rows.Err(); err != nil {
		return nil, fmt.Errorf("iterate questions: %w", err)
	}
	return result, nil
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
