package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// TicketHistoryRepository stores audit entries.
type TicketHistoryRepository interface {
	Create(ctx context.Context, history *domain.TicketHistory) error
	ListByTicket(ctx context.Context, ticketID string) ([]domain.TicketHistory, error)
}

type ticketHistoryRepository struct {
	pool *pgxpool.Pool
}

// NewTicketHistoryRepository builds repository.
func NewTicketHistoryRepository(pool *pgxpool.Pool) TicketHistoryRepository {
	return &ticketHistoryRepository{pool: pool}
}

func (r *ticketHistoryRepository) Create(ctx context.Context, history *domain.TicketHistory) error {
	const query = `
        INSERT INTO ticket_history (ticket_id, changed_by_type, changed_by_id, change_type, old_value, new_value)
        VALUES ($1,$2,$3,$4,$5,$6)
        RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		history.TicketID,
		history.ChangedByType,
		history.ChangedByID,
		history.ChangeType,
		history.OldValue,
		history.NewValue,
	).Scan(&history.ID, &history.CreatedAt)
}

func (r *ticketHistoryRepository) ListByTicket(ctx context.Context, ticketID string) ([]domain.TicketHistory, error) {
	const query = `
        SELECT id, ticket_id, changed_by_type, changed_by_id, change_type, old_value, new_value, created_at
        FROM ticket_history WHERE ticket_id=$1 ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.TicketHistory
	for rows.Next() {
		var history domain.TicketHistory
		if err := rows.Scan(
			&history.ID,
			&history.TicketID,
			&history.ChangedByType,
			&history.ChangedByID,
			&history.ChangeType,
			&history.OldValue,
			&history.NewValue,
			&history.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, history)
	}
	return result, rows.Err()
}
