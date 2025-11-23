package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// TeamRepository manages persistence for teams.
type TeamRepository interface {
	Create(ctx context.Context, team *domain.Team) error
	Update(ctx context.Context, team *domain.Team) error
	GetByID(ctx context.Context, id string) (*domain.Team, error)
	List(ctx context.Context, departmentID *string, includeInactive bool) ([]domain.Team, error)
}

type teamRepository struct {
	pool *pgxpool.Pool
}

// NewTeamRepository constructs repository.
func NewTeamRepository(pool *pgxpool.Pool) TeamRepository {
	return &teamRepository{pool: pool}
}

func (r *teamRepository) Create(ctx context.Context, team *domain.Team) error {
	const query = `
        INSERT INTO teams (department_id, name, description, is_active)
        VALUES ($1,$2,$3,$4)
        RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		team.DepartmentID,
		team.Name,
		team.Description,
		team.IsActive,
	).Scan(&team.ID, &team.CreatedAt, &team.UpdatedAt)
}

func (r *teamRepository) Update(ctx context.Context, team *domain.Team) error {
	const query = `
        UPDATE teams SET department_id=$1, name=$2, description=$3, is_active=$4, updated_at=NOW()
        WHERE id=$5`
	cmd, err := r.pool.Exec(ctx, query,
		team.DepartmentID,
		team.Name,
		team.Description,
		team.IsActive,
		team.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *teamRepository) GetByID(ctx context.Context, id string) (*domain.Team, error) {
	const query = `
        SELECT id, department_id, name, description, is_active, created_at, updated_at
        FROM teams WHERE id=$1`
	var team domain.Team
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&team.ID,
		&team.DepartmentID,
		&team.Name,
		&team.Description,
		&team.IsActive,
		&team.CreatedAt,
		&team.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &team, nil
}

func (r *teamRepository) List(ctx context.Context, departmentID *string, includeInactive bool) ([]domain.Team, error) {
	base := `
        SELECT id, department_id, name, description, is_active, created_at, updated_at
        FROM teams`
	args := []any{}
	clauses := []string{}
	if departmentID != nil {
		args = append(args, *departmentID)
		clauses = append(clauses, fmt.Sprintf("department_id=$%d", len(args)))
	}
	if !includeInactive {
		clauses = append(clauses, "is_active=TRUE")
	}
	if len(clauses) > 0 {
		base += " WHERE " + strings.Join(clauses, " AND ")
	}
	rows, err := r.pool.Query(ctx, base, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.Team
	for rows.Next() {
		var team domain.Team
		if err := rows.Scan(&team.ID, &team.DepartmentID, &team.Name, &team.Description, &team.IsActive, &team.CreatedAt, &team.UpdatedAt); err != nil {
			return nil, err
		}
		result = append(result, team)
	}
	return result, rows.Err()
}
