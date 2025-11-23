package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// DepartmentRepository manages department persistence.
type DepartmentRepository interface {
	Create(ctx context.Context, dept *domain.Department) error
	Update(ctx context.Context, dept *domain.Department) error
	GetByID(ctx context.Context, id string) (*domain.Department, error)
	ListActive(ctx context.Context) ([]domain.Department, error)
}

type departmentRepository struct {
	pool *pgxpool.Pool
}

// NewDepartmentRepository builds the repository.
func NewDepartmentRepository(pool *pgxpool.Pool) DepartmentRepository {
	return &departmentRepository{pool: pool}
}

func (r *departmentRepository) Create(ctx context.Context, dept *domain.Department) error {
	const query = `
        INSERT INTO departments (name, description, is_active)
        VALUES ($1,$2,$3)
        RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		dept.Name,
		dept.Description,
		dept.IsActive,
	).Scan(&dept.ID, &dept.CreatedAt, &dept.UpdatedAt)
}

func (r *departmentRepository) Update(ctx context.Context, dept *domain.Department) error {
	const query = `
        UPDATE departments SET name=$1, description=$2, is_active=$3, updated_at=NOW()
        WHERE id=$4`
	cmd, err := r.pool.Exec(ctx, query,
		dept.Name,
		dept.Description,
		dept.IsActive,
		dept.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *departmentRepository) GetByID(ctx context.Context, id string) (*domain.Department, error) {
	const query = `
        SELECT id, name, description, is_active, created_at, updated_at
        FROM departments WHERE id=$1`
	var dept domain.Department
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&dept.ID,
		&dept.Name,
		&dept.Description,
		&dept.IsActive,
		&dept.CreatedAt,
		&dept.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &dept, nil
}

func (r *departmentRepository) ListActive(ctx context.Context) ([]domain.Department, error) {
	const query = `
        SELECT id, name, description, is_active, created_at, updated_at
        FROM departments WHERE is_active = TRUE`
	rows, err := r.pool.Query(ctx, query)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Department
	for rows.Next() {
		var dept domain.Department
		if err := rows.Scan(&dept.ID, &dept.Name, &dept.Description, &dept.IsActive, &dept.CreatedAt, &dept.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, dept)
	}
	return result, rows.Err()
}
