package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/repository"
)

// TicketService coordinates ticket workflows.
type TicketService struct {
	tickets     repository.TicketRepository
	messages    repository.TicketMessageRepository
	attachments repository.AttachmentRepository
	departments repository.DepartmentRepository
	teams       repository.TeamRepository
	staff       repository.StaffRepository
}

// TicketDependencies bundles repositories for ticket service.
type TicketDependencies struct {
	TicketRepo     repository.TicketRepository
	MessageRepo    repository.TicketMessageRepository
	AttachmentRepo repository.AttachmentRepository
	DepartmentRepo repository.DepartmentRepository
	TeamRepo       repository.TeamRepository
	StaffRepo      repository.StaffRepository
}

// TicketCreateInput describes ticket creation payload.
type TicketCreateInput struct {
	DepartmentID string
	TeamID       *string
	Title        string
	Description  string
	Priority     domain.TicketPriority
	Tags         []string
}

// TicketUserFilter describes end-user listing filters.
type TicketUserFilter struct {
	Statuses    []domain.TicketStatus
	Priorities  []domain.TicketPriority
	CreatedFrom *time.Time
	CreatedTo   *time.Time
	Limit       int
	Offset      int
}

// TicketStaffFilter describes staff listing filters.
type TicketStaffFilter struct {
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

// MessageAttachmentInput defines attachment metadata.
type MessageAttachmentInput struct {
	StorageKey string
	FileName   string
	MimeType   string
	SizeBytes  int64
}

// NewTicketService constructs the service.
func NewTicketService(deps TicketDependencies) *TicketService {
	return &TicketService{
		tickets:     deps.TicketRepo,
		messages:    deps.MessageRepo,
		attachments: deps.AttachmentRepo,
		departments: deps.DepartmentRepo,
		teams:       deps.TeamRepo,
		staff:       deps.StaffRepo,
	}
}

// CreateTicket creates a ticket for a user.
func (s *TicketService) CreateTicket(ctx context.Context, userID string, input TicketCreateInput) (*domain.Ticket, error) {
	dept, err := s.departments.GetByID(ctx, input.DepartmentID)
	if err != nil {
		return nil, err
	}
	if !dept.IsActive {
		return nil, errors.New("department inactive")
	}
	if input.TeamID != nil {
		team, err := s.teams.GetByID(ctx, *input.TeamID)
		if err != nil {
			return nil, err
		}
		if !team.IsActive {
			return nil, errors.New("team inactive")
		}
		if team.DepartmentID != input.DepartmentID {
			return nil, errors.New("team not part of department")
		}
	}

	ticket := &domain.Ticket{
		ExternalKey:  generateTicketKey(),
		RequesterID:  userID,
		DepartmentID: input.DepartmentID,
		TeamID:       input.TeamID,
		Title:        strings.TrimSpace(input.Title),
		Description:  strings.TrimSpace(input.Description),
		Status:       domain.TicketStatusOpen,
		Priority:     input.Priority,
		Tags:         input.Tags,
	}

	if ticket.Priority == "" {
		ticket.Priority = domain.TicketPriorityMedium
	}

	if err := s.tickets.Create(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

// ListUserTickets returns paginated tickets for a requester.
func (s *TicketService) ListUserTickets(ctx context.Context, userID string, filter TicketUserFilter) ([]domain.Ticket, error) {
	repoFilter := repository.TicketFilter{
		RequesterID: &userID,
		Statuses:    filter.Statuses,
		Priorities:  filter.Priorities,
		CreatedFrom: filter.CreatedFrom,
		CreatedTo:   filter.CreatedTo,
		Limit:       filter.Limit,
		Offset:      filter.Offset,
	}
	return s.tickets.ListWithFilter(ctx, repoFilter)
}

// GetTicketForUser fetches a ticket ensuring ownership.
func (s *TicketService) GetTicketForUser(ctx context.Context, userID, ticketID string) (*domain.Ticket, []domain.TicketMessage, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if ticket.RequesterID != userID {
		return nil, nil, errors.New("access denied")
	}
	msgs, err := s.visibleMessagesForUser(ctx, ticket.ID)
	if err != nil {
		return nil, nil, err
	}
	return ticket, msgs, nil
}

// ListStaffTickets returns tickets accessible to staff.
func (s *TicketService) ListStaffTickets(ctx context.Context, staff *domain.StaffMember, filter TicketStaffFilter) ([]domain.Ticket, error) {
	repoFilter := repository.TicketFilter{
		DepartmentID: filter.DepartmentID,
		TeamID:       filter.TeamID,
		AssigneeID:   filter.AssigneeID,
		Statuses:     filter.Statuses,
		Priorities:   filter.Priorities,
		SearchTerm:   filter.SearchTerm,
		CreatedFrom:  filter.CreatedFrom,
		CreatedTo:    filter.CreatedTo,
		UpdatedFrom:  filter.UpdatedFrom,
		UpdatedTo:    filter.UpdatedTo,
		Limit:        filter.Limit,
		Offset:       filter.Offset,
	}
	s.applyStaffScope(&repoFilter, staff)
	return s.tickets.ListWithFilter(ctx, repoFilter)
}

// GetTicketForStaff fetches ticket ensuring staff access.
func (s *TicketService) GetTicketForStaff(ctx context.Context, staff *domain.StaffMember, ticketID string) (*domain.Ticket, []domain.TicketMessage, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, nil, err
	}
	if !s.staffCanAccessTicket(staff, ticket) {
		return nil, nil, errors.New("access denied")
	}
	msgs, err := s.messagesWithAttachments(ctx, ticket.ID)
	if err != nil {
		return nil, nil, err
	}
	return ticket, msgs, nil
}

// AddMessage appends a message to a ticket.
func (s *TicketService) AddMessage(ctx context.Context, actor domain.SubjectType, actorID string, staff *domain.StaffMember, ticketID string, messageType domain.TicketMessageType, body string, attachments []MessageAttachmentInput) (*domain.TicketMessage, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	switch actor {
	case domain.SubjectTypeUser:
		if ticket.RequesterID != actorID {
			return nil, errors.New("access denied")
		}
		if messageType != domain.MessageTypePublicReply {
			return nil, errors.New("users can only post public replies")
		}
	case domain.SubjectTypeStaff:
		if staff == nil {
			return nil, errors.New("staff context required")
		}
		if !s.staffCanAccessTicket(staff, ticket) {
			return nil, errors.New("access denied")
		}
		if messageType != domain.MessageTypePublicReply && messageType != domain.MessageTypeInternalNote {
			return nil, errors.New("invalid message type for staff")
		}
	default:
		return nil, errors.New("unknown actor")
	}

	msg := &domain.TicketMessage{
		TicketID:    ticket.ID,
		MessageType: messageType,
		Body:        strings.TrimSpace(body),
	}
	if actor == domain.SubjectTypeUser {
		msg.AuthorType = domain.AuthorTypeUser
		authorID := ticket.RequesterID
		msg.AuthorID = &authorID
	} else {
		msg.AuthorType = domain.AuthorTypeStaff
		if staff != nil {
			msg.AuthorID = &staff.ID
		}
	}

	if err := s.messages.Create(ctx, msg); err != nil {
		return nil, err
	}
	for _, att := range attachments {
		record := &domain.AttachmentReference{
			TicketMessageID: msg.ID,
			StorageKey:      att.StorageKey,
			FileName:        att.FileName,
			MimeType:        att.MimeType,
			SizeBytes:       att.SizeBytes,
		}
		if err := s.attachments.Create(ctx, record); err != nil {
			return nil, err
		}
		msg.Attachments = append(msg.Attachments, *record)
	}
	return msg, nil
}

// CloseTicketAsUser closes ticket when allowed states.
func (s *TicketService) CloseTicketAsUser(ctx context.Context, userID, ticketID string) (*domain.Ticket, error) {
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.RequesterID != userID {
		return nil, errors.New("access denied")
	}
	if ticket.Status != domain.TicketStatusResolved && ticket.Status != domain.TicketStatusPendingUser {
		return nil, errors.New("ticket cannot be closed in current status")
	}
	now := time.Now()
	ticket.Status = domain.TicketStatusClosed
	ticket.ClosedAt = &now
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}
	return ticket, nil
}

func (s *TicketService) applyStaffScope(filter *repository.TicketFilter, staff *domain.StaffMember) {
	if staff == nil || staff.Role == domain.StaffRoleAdmin {
		return
	}
	if staff.DepartmentID != nil {
		filter.DepartmentID = staff.DepartmentID
	}
	if staff.TeamID != nil {
		filter.TeamID = staff.TeamID
	}
}

func (s *TicketService) staffCanAccessTicket(staff *domain.StaffMember, ticket *domain.Ticket) bool {
	if staff == nil {
		return false
	}
	if staff.Role == domain.StaffRoleAdmin {
		return true
	}
	if staff.TeamID != nil && ticket.TeamID != nil && *staff.TeamID == *ticket.TeamID {
		return true
	}
	if staff.DepartmentID != nil && *staff.DepartmentID == ticket.DepartmentID {
		return true
	}
	return false
}

func (s *TicketService) visibleMessagesForUser(ctx context.Context, ticketID string) ([]domain.TicketMessage, error) {
	msgs, err := s.messagesWithAttachments(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	filtered := make([]domain.TicketMessage, 0, len(msgs))
	for _, msg := range msgs {
		if msg.MessageType == domain.MessageTypeInternalNote {
			continue
		}
		filtered = append(filtered, msg)
	}
	return filtered, nil
}

func (s *TicketService) messagesWithAttachments(ctx context.Context, ticketID string) ([]domain.TicketMessage, error) {
	msgs, err := s.messages.ListByTicket(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	for i := range msgs {
		attachments, err := s.attachments.ListByMessage(ctx, msgs[i].ID)
		if err != nil {
			return nil, err
		}
		msgs[i].Attachments = attachments
	}
	return msgs, nil
}

func generateTicketKey() string {
	return "TCK-" + strings.ToUpper(strings.ReplaceAll(uuid.NewString(), "-", "")[:8])
}
