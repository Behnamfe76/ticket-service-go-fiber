package repository

import (
	"context"

	"github.com/jackc/pgx/v5/pgxpool"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// AttachmentRepository persists attachment metadata.
type AttachmentRepository interface {
	Create(ctx context.Context, attachment *domain.AttachmentReference) error
	ListByMessage(ctx context.Context, messageID string) ([]domain.AttachmentReference, error)
}

type attachmentRepository struct {
	pool *pgxpool.Pool
}

// NewAttachmentRepository constructs repository.
func NewAttachmentRepository(pool *pgxpool.Pool) AttachmentRepository {
	return &attachmentRepository{pool: pool}
}

func (r *attachmentRepository) Create(ctx context.Context, attachment *domain.AttachmentReference) error {
	const query = `
        INSERT INTO attachment_references (ticket_message_id, storage_key, file_name, mime_type, size_bytes)
        VALUES ($1,$2,$3,$4,$5)
        RETURNING id, created_at`
	return r.pool.QueryRow(ctx, query,
		attachment.TicketMessageID,
		attachment.StorageKey,
		attachment.FileName,
		attachment.MimeType,
		attachment.SizeBytes,
	).Scan(&attachment.ID, &attachment.CreatedAt)
}

func (r *attachmentRepository) ListByMessage(ctx context.Context, messageID string) ([]domain.AttachmentReference, error) {
	const query = `
        SELECT id, ticket_message_id, storage_key, file_name, mime_type, size_bytes, created_at
        FROM attachment_references WHERE ticket_message_id=$1`
	rows, err := r.pool.Query(ctx, query, messageID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var result []domain.AttachmentReference
	for rows.Next() {
		var attachment domain.AttachmentReference
		if err := rows.Scan(
			&attachment.ID,
			&attachment.TicketMessageID,
			&attachment.StorageKey,
			&attachment.FileName,
			&attachment.MimeType,
			&attachment.SizeBytes,
			&attachment.CreatedAt,
		); err != nil {
			return nil, err
		}
		result = append(result, attachment)
	}
	return result, rows.Err()
}
