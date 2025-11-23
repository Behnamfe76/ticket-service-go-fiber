package repository

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

// PasswordResetToken represents stored reset tokens.
type PasswordResetToken struct {
	ID          string
	SubjectType string
	SubjectID   string
	Token       string
	ExpiresAt   time.Time
	UsedAt      *time.Time
	CreatedAt   time.Time
}

// PasswordResetRepository manages password reset token persistence.
type PasswordResetRepository interface {
	Create(ctx context.Context, token *PasswordResetToken) error
	GetByToken(ctx context.Context, token string) (*PasswordResetToken, error)
	MarkUsed(ctx context.Context, id string) error
}

type passwordResetRepository struct {
	pool *pgxpool.Pool
}

// NewPasswordResetRepository constructs repository.
func NewPasswordResetRepository(pool *pgxpool.Pool) PasswordResetRepository {
	return &passwordResetRepository{pool: pool}
}

func (r *passwordResetRepository) Create(ctx context.Context, token *PasswordResetToken) error {
	const query = `
        INSERT INTO password_reset_tokens (subject_type, subject_id, token, expires_at)
        VALUES ($1,$2,$3,$4)
        RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		token.SubjectType,
		token.SubjectID,
		token.Token,
		token.ExpiresAt,
	).Scan(&token.ID, &token.CreatedAt)
}

func (r *passwordResetRepository) GetByToken(ctx context.Context, tokenStr string) (*PasswordResetToken, error) {
	const query = `
        SELECT id, subject_type, subject_id, token, expires_at, used_at, created_at
        FROM password_reset_tokens WHERE token=$1`
	var token PasswordResetToken
	if err := r.pool.QueryRow(ctx, query, tokenStr).Scan(
		&token.ID,
		&token.SubjectType,
		&token.SubjectID,
		&token.Token,
		&token.ExpiresAt,
		&token.UsedAt,
		&token.CreatedAt,
	); err != nil {
		return nil, err
	}
	return &token, nil
}

func (r *passwordResetRepository) MarkUsed(ctx context.Context, id string) error {
	const query = `
        UPDATE password_reset_tokens SET used_at=NOW()
        WHERE id=$1`
	_, err := r.pool.Exec(ctx, query, id)
	return err
}
