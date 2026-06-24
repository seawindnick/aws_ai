package repository

import (
	"context"
	"database/sql"
	"fmt"
	"strings"
	"time"

	"github.com/workshop/wrong-question/internal/model"
)

type UserRepo struct {
	db *sql.DB
}

func NewUserRepo(db *sql.DB) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	_, err := r.db.ExecContext(ctx,
		`INSERT INTO users (id, cognito_sub, email, nickname, role, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		u.ID, u.CognitoSub, u.Email, u.Nickname, string(u.Role), string(u.Status), u.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByID(ctx context.Context, id string) (*model.User, error) {
	return r.scanOne(r.db.QueryRowContext(ctx,
		`SELECT id, cognito_sub, email, nickname, role, status, deactivated_at, created_at
		 FROM users WHERE id = ?`, id,
	))
}

func (r *UserRepo) GetByCognitoSub(ctx context.Context, sub string) (*model.User, error) {
	return r.scanOne(r.db.QueryRowContext(ctx,
		`SELECT id, cognito_sub, email, nickname, role, status, deactivated_at, created_at
		 FROM users WHERE cognito_sub = ?`, sub,
	))
}

func (r *UserRepo) UpdateNickname(ctx context.Context, id, nickname string) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET nickname = ? WHERE id = ?`, nickname, id)
	return err
}

func (r *UserRepo) UpdateStatus(ctx context.Context, id string, status model.UserStatus, deactivatedAt *time.Time) error {
	_, err := r.db.ExecContext(ctx,
		`UPDATE users SET status = ?, deactivated_at = ? WHERE id = ?`,
		string(status), deactivatedAt, id,
	)
	return err
}

func (r *UserRepo) UpdateRole(ctx context.Context, id string, role model.Role) error {
	_, err := r.db.ExecContext(ctx, `UPDATE users SET role = ? WHERE id = ?`, string(role), id)
	return err
}

func (r *UserRepo) List(ctx context.Context, role, status string, page, pageSize int) ([]*model.User, error) {
	offset := (page - 1) * pageSize
	where := []string{"1=1"}
	args := []any{}
	if role != "" {
		where = append(where, "role = ?")
		args = append(args, role)
	}
	if status != "" {
		where = append(where, "status = ?")
		args = append(args, status)
	}
	args = append(args, pageSize, offset)
	rows, err := r.db.QueryContext(ctx,
		`SELECT id, cognito_sub, email, nickname, role, status, deactivated_at, created_at
		 FROM users WHERE `+strings.Join(where, " AND ")+
			` ORDER BY created_at DESC LIMIT ? OFFSET ?`,
		args...,
	)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	defer rows.Close()
	return r.scanMany(rows)
}

func (r *UserRepo) scanOne(row *sql.Row) (*model.User, error) {
	u := &model.User{}
	var deactivatedAt sql.NullTime
	err := row.Scan(&u.ID, &u.CognitoSub, &u.Email, &u.Nickname, &u.Role, &u.Status, &deactivatedAt, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("scan user: %w", err)
	}
	if deactivatedAt.Valid {
		t := deactivatedAt.Time
		u.DeactivatedAt = &t
	}
	return u, nil
}

func (r *UserRepo) scanMany(rows *sql.Rows) ([]*model.User, error) {
	var result []*model.User
	for rows.Next() {
		u := &model.User{}
		var deactivatedAt sql.NullTime
		if err := rows.Scan(&u.ID, &u.CognitoSub, &u.Email, &u.Nickname, &u.Role, &u.Status, &deactivatedAt, &u.CreatedAt); err != nil {
			return nil, fmt.Errorf("scan user row: %w", err)
		}
		if deactivatedAt.Valid {
			t := deactivatedAt.Time
			u.DeactivatedAt = &t
		}
		result = append(result, u)
	}
	return result, rows.Err()
}
