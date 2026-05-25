package repository

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/evadeplayer/api/internal/model"
)

type UserRepo struct {
	db *pgxpool.Pool
}

func NewUserRepo(db *pgxpool.Pool) *UserRepo {
	return &UserRepo{db: db}
}

func (r *UserRepo) Create(ctx context.Context, u *model.User) error {
	q := `INSERT INTO users (email, password) VALUES ($1, $2)
	      RETURNING id, created_at`
	err := r.db.QueryRow(ctx, q, u.Email, u.Password).Scan(&u.ID, &u.CreatedAt)
	if err != nil {
		if strings.Contains(err.Error(), "23505") {
			return ErrEmailTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	return nil
}

func (r *UserRepo) FindByEmail(ctx context.Context, email string) (*model.User, error) {
	u := &model.User{}
	q := `SELECT id, email, password, created_at FROM users WHERE email = $1`
	err := r.db.QueryRow(ctx, q, email).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by email: %w", err)
	}
	return u, nil
}

func (r *UserRepo) FindByID(ctx context.Context, id string) (*model.User, error) {
	u := &model.User{}
	q := `SELECT id, email, password, created_at FROM users WHERE id = $1`
	err := r.db.QueryRow(ctx, q, id).Scan(&u.ID, &u.Email, &u.Password, &u.CreatedAt)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, ErrNotFound
		}
		return nil, fmt.Errorf("find user by id: %w", err)
	}
	return u, nil
}
