package repository

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/workshop/wrong-question/internal/model"
)

type ClassRepo struct {
	db *sql.DB
}

func NewClassRepo(db *sql.DB) *ClassRepo {
	return &ClassRepo{db: db}
}

func (r *ClassRepo) Create(ctx context.Context, c *model.Class) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO classes (id, name, teacher_id, invite_code, created_at)
		 VALUES (?, ?, ?, ?, ?)`,
		c.ID, c.Name, c.TeacherID, c.InviteCode, c.CreatedAt,
	)
	return err
}

func (r *ClassRepo) GetByID(ctx context.Context, id string) (*model.Class, error) {
	c := &model.Class{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, teacher_id, invite_code, created_at FROM classes WHERE id = ?`, id,
	).Scan(&c.ID, &c.Name, &c.TeacherID, &c.InviteCode, &c.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get class: %w", err)
	}
	return c, nil
}

func (r *ClassRepo) GetByInviteCode(ctx context.Context, code string) (*model.Class, error) {
	c := &model.Class{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, name, teacher_id, invite_code, created_at FROM classes WHERE invite_code = ?`, code,
	).Scan(&c.ID, &c.Name, &c.TeacherID, &c.InviteCode, &c.CreatedAt)
	if err == sql.ErrNoRows {
		return nil, nil
	}
	if err != nil {
		return nil, fmt.Errorf("get class by invite code: %w", err)
	}
	return c, nil
}

func (r *ClassRepo) ListByTeacher(ctx context.Context, teacherID string) ([]*model.Class, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, name, teacher_id, invite_code, created_at FROM classes WHERE teacher_id = ? ORDER BY created_at DESC`,
		teacherID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanMany(rows)
}

func (r *ClassRepo) ListByMember(ctx context.Context, userID string) ([]*model.Class, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT c.id, c.name, c.teacher_id, c.invite_code, c.created_at
		 FROM classes c JOIN class_members cm ON c.id = cm.class_id
		 WHERE cm.user_id = ? ORDER BY cm.joined_at DESC`,
		userID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return r.scanMany(rows)
}

func (r *ClassRepo) UpdateInviteCode(ctx context.Context, classID, code string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE classes SET invite_code = ? WHERE id = ?`, code, classID)
	return err
}

func (r *ClassRepo) AddMember(ctx context.Context, m *model.ClassMember) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT IGNORE INTO class_members (id, class_id, user_id, joined_at) VALUES (?, ?, ?, ?)`,
		m.ID, m.ClassID, m.UserID, m.JoinedAt,
	)
	return err
}

func (r *ClassRepo) RemoveMember(ctx context.Context, classID, userID string) error {
	_, err := r.db.ExecContext(ctx,
		`DELETE FROM class_members WHERE class_id = ? AND user_id = ?`, classID, userID,
	)
	return err
}

func (r *ClassRepo) IsMember(ctx context.Context, classID, userID string) (bool, error) {
	var count int
	err := r.db.QueryRowContext(ctx,
		`SELECT COUNT(*) FROM class_members WHERE class_id = ? AND user_id = ?`, classID, userID,
	).Scan(&count)
	return count > 0, err
}

func (r *ClassRepo) ListMembers(ctx context.Context, classID string) ([]*model.ClassMember, error) {
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, class_id, user_id, joined_at FROM class_members WHERE class_id = ? ORDER BY joined_at`,
		classID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var result []*model.ClassMember
	for rows.Next() {
		m := &model.ClassMember{}
		if err := rows.Scan(&m.ID, &m.ClassID, &m.UserID, &m.JoinedAt); err != nil {
			return nil, err
		}
		result = append(result, m)
	}
	return result, rows.Err()
}

func (r *ClassRepo) scanMany(rows *sql.Rows) ([]*model.Class, error) {
	var result []*model.Class
	for rows.Next() {
		c := &model.Class{}
		if err := rows.Scan(&c.ID, &c.Name, &c.TeacherID, &c.InviteCode, &c.CreatedAt); err != nil {
			return nil, err
		}
		result = append(result, c)
	}
	return result, rows.Err()
}
