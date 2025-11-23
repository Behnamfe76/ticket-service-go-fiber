package repository

import (
	"context"
	"fmt"
	"strings"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// StaffRepository handles persistence for staff members.
type StaffRepository interface {
	Create(ctx context.Context, staff *domain.StaffMember) error
	Update(ctx context.Context, staff *domain.StaffMember) error
	GetByID(ctx context.Context, id string) (*domain.StaffMember, error)
	GetByEmail(ctx context.Context, email string) (*domain.StaffMember, error)
	List(ctx context.Context, filter StaffFilter) ([]domain.StaffMember, error)
}

// StaffFilter defines query params for staff listing.
type StaffFilter struct {
	Role         *domain.StaffRole
	TeamID       *string
	DepartmentID *string
	Active       *bool
	Limit        int
	Offset       int
}

type staffRepository struct {
	pool *pgxpool.Pool
}

// NewStaffRepository instantiates the repository.
func NewStaffRepository(pool *pgxpool.Pool) StaffRepository {
	return &staffRepository{pool: pool}
}

func (r *staffRepository) Create(ctx context.Context, staff *domain.StaffMember) error {
	const query = `
        INSERT INTO staff_members (name, email, password_hash, role, department_id, team_id, active_flag)
        VALUES ($1,$2,$3,$4,$5,$6,$7)
        RETURNING id, created_at, updated_at`

	return r.pool.QueryRow(ctx, query,
		staff.Name,
		staff.Email,
		staff.PasswordHash,
		staff.Role,
		staff.DepartmentID,
		staff.TeamID,
		staff.Active,
	).Scan(&staff.ID, &staff.CreatedAt, &staff.UpdatedAt)
}

func (r *staffRepository) Update(ctx context.Context, staff *domain.StaffMember) error {
	const query = `
        UPDATE staff_members
        SET name=$1, email=$2, password_hash=$3, role=$4, department_id=$5, team_id=$6, active_flag=$7, updated_at=NOW()
        WHERE id=$8`

	cmd, err := r.pool.Exec(ctx, query,
		staff.Name,
		staff.Email,
		staff.PasswordHash,
		staff.Role,
		staff.DepartmentID,
		staff.TeamID,
		staff.Active,
		staff.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *staffRepository) GetByID(ctx context.Context, id string) (*domain.StaffMember, error) {
	const query = `
        SELECT id, name, email, password_hash, role, department_id, team_id, active_flag, created_at, updated_at
        FROM staff_members WHERE id=$1`

	var staff domain.StaffMember
	if err := r.pool.QueryRow(ctx, query, id).Scan(
		&staff.ID,
		&staff.Name,
		&staff.Email,
		&staff.PasswordHash,
		&staff.Role,
		&staff.DepartmentID,
		&staff.TeamID,
		&staff.Active,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &staff, nil
}

func (r *staffRepository) GetByEmail(ctx context.Context, email string) (*domain.StaffMember, error) {
	const query = `
        SELECT id, name, email, password_hash, role, department_id, team_id, active_flag, created_at, updated_at
        FROM staff_members WHERE email=$1`

	var staff domain.StaffMember
	if err := r.pool.QueryRow(ctx, query, email).Scan(
		&staff.ID,
		&staff.Name,
		&staff.Email,
		&staff.PasswordHash,
		&staff.Role,
		&staff.DepartmentID,
		&staff.TeamID,
		&staff.Active,
		&staff.CreatedAt,
		&staff.UpdatedAt,
	); err != nil {
		return nil, err
	}
	return &staff, nil
}

func (r *staffRepository) List(ctx context.Context, filter StaffFilter) ([]domain.StaffMember, error) {
	query := `
        SELECT id, name, email, password_hash, role, department_id, team_id, active_flag, created_at, updated_at
        FROM staff_members`
	args := []any{}
	clauses := []string{}

	if filter.Role != nil {
		args = append(args, *filter.Role)
		clauses = append(clauses, fmt.Sprintf("role=$%d", len(args)))
	}
	if filter.TeamID != nil {
		args = append(args, *filter.TeamID)
		clauses = append(clauses, fmt.Sprintf("team_id=$%d", len(args)))
	}
	if filter.DepartmentID != nil {
		args = append(args, *filter.DepartmentID)
		clauses = append(clauses, fmt.Sprintf("department_id=$%d", len(args)))
	}
	if filter.Active != nil {
		args = append(args, *filter.Active)
		clauses = append(clauses, fmt.Sprintf("active_flag=$%d", len(args)))
	}
	if len(clauses) > 0 {
		query += " WHERE " + strings.Join(clauses, " AND ")
	}

	query += " ORDER BY created_at DESC"
	limit := filter.Limit
	if limit <= 0 {
		limit = 50
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}
	query += fmt.Sprintf(" LIMIT %d OFFSET %d", limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.StaffMember
	for rows.Next() {
		var staff domain.StaffMember
		if err := rows.Scan(
			&staff.ID,
			&staff.Name,
			&staff.Email,
			&staff.PasswordHash,
			&staff.Role,
			&staff.DepartmentID,
			&staff.TeamID,
			&staff.Active,
			&staff.CreatedAt,
			&staff.UpdatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, staff)
	}
	return result, rows.Err()
}
