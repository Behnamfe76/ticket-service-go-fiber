package repository

import (
	"context"

	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// TicketMessageRepository manages ticket thread messages.
type TicketMessageRepository interface {
	Create(ctx context.Context, msg *domain.TicketMessage) error
	ListByTicket(ctx context.Context, ticketID string) ([]domain.TicketMessage, error)
}

type ticketMessageRepository struct {
	pool *pgxpool.Pool
}

// NewTicketMessageRepository builds repository.
func NewTicketMessageRepository(pool *pgxpool.Pool) TicketMessageRepository {
	return &ticketMessageRepository{pool: pool}
}

func (r *ticketMessageRepository) Create(ctx context.Context, msg *domain.TicketMessage) error {
	const query = `
        INSERT INTO ticket_messages (ticket_id, author_type, author_id, message_type, body)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		msg.TicketID,
		msg.AuthorType,
		msg.AuthorID,
		msg.MessageType,
		msg.Body,
	).Scan(&msg.ID, &msg.CreatedAt)
}

func (r *ticketMessageRepository) ListByTicket(ctx context.Context, ticketID string) ([]domain.TicketMessage, error) {
	const query = `
        SELECT id, ticket_id, author_type, author_id, message_type, body, created_at
        FROM ticket_messages WHERE ticket_id=$1 ORDER BY created_at ASC`
	rows, err := r.pool.Query(ctx, query, ticketID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.TicketMessage
	for rows.Next() {
		var msg domain.TicketMessage
		if err := rows.Scan(
			&msg.ID,
			&msg.TicketID,
			&msg.AuthorType,
			&msg.AuthorID,
			&msg.MessageType,
			&msg.Body,
			&msg.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, msg)
	}
	return result, rows.Err()
}
