package repository

import (
	"context"
	"database/sql"
	"fmt"

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
		`INSERT INTO users (id, cognito_sub, email, role, school_name, created_at)
		 VALUES (?, ?, ?, ?, ?, ?)`,
		u.ID, u.CognitoSub, u.Email, u.Role, u.SchoolName, u.CreatedAt,
	)
	if err != nil {
		return fmt.Errorf("insert user: %w", err)
	}
	return nil
}

func (r *UserRepo) GetByCognitoSub(ctx context.Context, sub string) (*model.User, error) {
	u := &model.User{}
	err := r.db.QueryRowContext(ctx,
		`SELECT id, cognito_sub, email, role, school_name, created_at
		 FROM users WHERE cognito_sub = ?`, sub,
	).Scan(&u.ID, &u.CognitoSub, &u.Email, &u.Role, &u.SchoolName, &u.CreatedAt)
	if err != nil {
		return nil, fmt.Errorf("get user by cognito_sub: %w", err)
	}
	return u, nil
}
