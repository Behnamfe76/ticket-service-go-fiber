package repository

import (
	"context"
	"fmt"
	"strings"
	"time"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// TicketFilter captures staff search parameters.
type TicketFilter struct {
	RequesterID  *string
	DepartmentID *string
	TeamID       *string
	AssigneeID   *string
	Statuses     []domain.TicketStatus
	Priorities   []domain.TicketPriority
	SearchTerm   *string
	CreatedFrom  *time.Time
	CreatedTo    *time.Time
	UpdatedFrom  *time.Time
	UpdatedTo    *time.Time
	Limit        int
	Offset       int
}

// TicketRepository encapsulates ticket persistence.
type TicketRepository interface {
	Create(ctx context.Context, ticket *domain.Ticket) error
	Update(ctx context.Context, ticket *domain.Ticket) error
	GetByID(ctx context.Context, id string) (*domain.Ticket, error)
	GetByExternalKey(ctx context.Context, key string) (*domain.Ticket, error)
	ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Ticket, error)
	ListWithFilter(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error)
}

type ticketRepository struct {
	pool *pgxpool.Pool
}

// NewTicketRepository instantiates repository.
func NewTicketRepository(pool *pgxpool.Pool) TicketRepository {
	return &ticketRepository{pool: pool}
}

func (r *ticketRepository) Create(ctx context.Context, ticket *domain.Ticket) error {
	const query = `
        INSERT INTO tickets (external_key, requester_user_id, department_id, team_id, assignee_staff_id, title, description, status, priority, tags)
        VALUES ($1,$2,$3,$4,$5,$6,$7,$8,$9,$10)
        RETURNING id, created_at, updated_at`
	return r.pool.QueryRow(ctx, query,
		ticket.ExternalKey,
		ticket.RequesterID,
		ticket.DepartmentID,
		ticket.TeamID,
		ticket.AssigneeID,
		ticket.Title,
		ticket.Description,
		ticket.Status,
		ticket.Priority,
		ticket.Tags,
	).Scan(&ticket.ID, &ticket.CreatedAt, &ticket.UpdatedAt)
}

func (r *ticketRepository) Update(ctx context.Context, ticket *domain.Ticket) error {
	const query = `
        UPDATE tickets SET department_id=$1, team_id=$2, assignee_staff_id=$3, title=$4, description=$5,
            status=$6, priority=$7, tags=$8, closed_at=$9, updated_at=NOW()
        WHERE id=$10`
	cmd, err := r.pool.Exec(ctx, query,
		ticket.DepartmentID,
		ticket.TeamID,
		ticket.AssigneeID,
		ticket.Title,
		ticket.Description,
		ticket.Status,
		ticket.Priority,
		ticket.Tags,
		ticket.ClosedAt,
		ticket.ID,
	)
	if err != nil {
		return err
	}
	if cmd.RowsAffected() == 0 {
		return pgx.ErrNoRows
	}
	return nil
}

func (r *ticketRepository) GetByID(ctx context.Context, id string) (*domain.Ticket, error) {
	const query = `
        SELECT id, external_key, requester_user_id, department_id, team_id, assignee_staff_id,
               title, description, status, priority, tags, created_at, updated_at, closed_at
        FROM tickets WHERE id=$1`
	return r.fetchSingle(ctx, query, id)
}

func (r *ticketRepository) GetByExternalKey(ctx context.Context, key string) (*domain.Ticket, error) {
	const query = `
        SELECT id, external_key, requester_user_id, department_id, team_id, assignee_staff_id,
               title, description, status, priority, tags, created_at, updated_at, closed_at
        FROM tickets WHERE external_key=$1`
	return r.fetchSingle(ctx, query, key)
}

func (r *ticketRepository) fetchSingle(ctx context.Context, query string, arg any) (*domain.Ticket, error) {
	var ticket domain.Ticket
	if err := r.pool.QueryRow(ctx, query, arg).Scan(
		&ticket.ID,
		&ticket.ExternalKey,
		&ticket.RequesterID,
		&ticket.DepartmentID,
		&ticket.TeamID,
		&ticket.AssigneeID,
		&ticket.Title,
		&ticket.Description,
		&ticket.Status,
		&ticket.Priority,
		&ticket.Tags,
		&ticket.CreatedAt,
		&ticket.UpdatedAt,
		&ticket.ClosedAt,
	); err != nil {
		return nil, err
	}
	return &ticket, nil
}

func (r *ticketRepository) ListByUser(ctx context.Context, userID string, limit, offset int) ([]domain.Ticket, error) {
	filter := TicketFilter{
		RequesterID: &userID,
		Limit:       limit,
		Offset:      offset,
	}
	return r.ListWithFilter(ctx, filter)
}

func (r *ticketRepository) ListWithFilter(ctx context.Context, filter TicketFilter) ([]domain.Ticket, error) {
	base := `SELECT id, external_key, requester_user_id, department_id, team_id, assignee_staff_id,
                    title, description, status, priority, tags, created_at, updated_at, closed_at
             FROM tickets`
	clauses := []string{"1=1"}
	args := []any{}

	if filter.RequesterID != nil {
		args = append(args, *filter.RequesterID)
		clauses = append(clauses, fmt.Sprintf("requester_user_id=$%d", len(args)))
	}
	if filter.DepartmentID != nil {
		args = append(args, *filter.DepartmentID)
		clauses = append(clauses, fmt.Sprintf("department_id=$%d", len(args)))
	}
	if filter.TeamID != nil {
		args = append(args, *filter.TeamID)
		clauses = append(clauses, fmt.Sprintf("team_id=$%d", len(args)))
	}
	if filter.AssigneeID != nil {
		args = append(args, *filter.AssigneeID)
		clauses = append(clauses, fmt.Sprintf("assignee_staff_id=$%d", len(args)))
	}
	if len(filter.Statuses) > 0 {
		placeholders := make([]string, len(filter.Statuses))
		for i, status := range filter.Statuses {
			args = append(args, status)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		clauses = append(clauses, fmt.Sprintf("status IN (%s)", strings.Join(placeholders, ",")))
	}
	if len(filter.Priorities) > 0 {
		placeholders := make([]string, len(filter.Priorities))
		for i, pr := range filter.Priorities {
			args = append(args, pr)
			placeholders[i] = fmt.Sprintf("$%d", len(args))
		}
		clauses = append(clauses, fmt.Sprintf("priority IN (%s)", strings.Join(placeholders, ",")))
	}
	if filter.CreatedFrom != nil {
		args = append(args, *filter.CreatedFrom)
		clauses = append(clauses, fmt.Sprintf("created_at >= $%d", len(args)))
	}
	if filter.CreatedTo != nil {
		args = append(args, *filter.CreatedTo)
		clauses = append(clauses, fmt.Sprintf("created_at <= $%d", len(args)))
	}
	if filter.UpdatedFrom != nil {
		args = append(args, *filter.UpdatedFrom)
		clauses = append(clauses, fmt.Sprintf("updated_at >= $%d", len(args)))
	}
	if filter.UpdatedTo != nil {
		args = append(args, *filter.UpdatedTo)
		clauses = append(clauses, fmt.Sprintf("updated_at <= $%d", len(args)))
	}
	if filter.SearchTerm != nil && strings.TrimSpace(*filter.SearchTerm) != "" {
		search := "%" + strings.ToLower(strings.TrimSpace(*filter.SearchTerm)) + "%"
		args = append(args, search)
		placeholder := fmt.Sprintf("$%d", len(args))
		clauses = append(clauses, fmt.Sprintf("(LOWER(title) LIKE %s OR LOWER(description) LIKE %s)", placeholder, placeholder))
	}

	limit := filter.Limit
	if limit <= 0 {
		limit = 20
	}
	offset := filter.Offset
	if offset < 0 {
		offset = 0
	}

	query := fmt.Sprintf(`%s WHERE %s ORDER BY updated_at DESC LIMIT %d OFFSET %d`,
		base, strings.Join(clauses, " AND "), limit, offset)

	rows, err := r.pool.Query(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	return scanTickets(rows)
}

func scanTickets(rows pgx.Rows) ([]domain.Ticket, error) {
	var result []domain.Ticket
	for rows.Next() {
		var ticket domain.Ticket
		if err := rows.Scan(
			&ticket.ID,
			&ticket.ExternalKey,
			&ticket.RequesterID,
			&ticket.DepartmentID,
			&ticket.TeamID,
			&ticket.AssigneeID,
			&ticket.Title,
			&ticket.Description,
			&ticket.Status,
			&ticket.Priority,
			&ticket.Tags,
			&ticket.CreatedAt,
			&ticket.UpdatedAt,
			&ticket.ClosedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, ticket)
	}
	return result, rows.Err()
}
