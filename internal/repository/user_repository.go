package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// UserRepository defines persistence access for end-users.
type UserRepository interface {
	Create(ctx context.Context, user *domain.User) error
	Update(ctx context.Context, user *domain.User) error
	GetByID(ctx context.Context, id string) (*domain.User, error)
	GetByEmail(ctx context.Context, email string) (*domain.User, error)
}

type userRepository struct {
	pool *pgxpool.Pool
}

// NewUserRepository returns a Postgres-backed implementation.
func NewUserRepository(pool *pgxpool.Pool) UserRepository {
	return &userRepository{pool: pool}
}

func (r *userRepository) Create(ctx context.Context, user *domain.User) error {
	const query = `
        INSERT INTO users (name, email, password_hash, status)
        VALUES ($1, $2, $3, $4)
        RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Status,
	).Scan(&user.ID, &user.CreatedAt, &user.UpdatedAt)
}

func (r *userRepository) Update(ctx context.Context, user *domain.User) error {
	const query = `
        UPDATE users SET name=$1, email=$2, password_hash=$3, status=$4, updated_at=NOW()
        WHERE id=$5`

	cmd, err := r.pool.Exec(ctx, query,
		user.Name,
		user.Email,
		user.PasswordHash,
		user.Status,
		user.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *userRepository) GetByID(ctx context.Context, id string) (*domain.User, error) {
	const query = `
        SELECT id, name, email, password_hash, status, created_at, updated_at
        FROM users WHERE id=$1`

	var user domain.User
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}

func (r *userRepository) GetByEmail(ctx context.Context, email string) (*domain.User, error) {
	const query = `
        SELECT id, name, email, password_hash, status, created_at, updated_at
        FROM users WHERE email=$1`

	var user domain.User
	if err := r.pool.QueryRow(ctx, query, email).Scan(
		&user.ID,
		&user.Name,
		&user.Email,
		&user.PasswordHash,
		&user.Status,
		&user.CreatedAt,
		&user.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &user, nil
}
