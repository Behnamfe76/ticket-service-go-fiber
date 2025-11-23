package service

import (
	"context"
	"errors"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/events"
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
	history     repository.TicketHistoryRepository
	dispatcher  events.Dispatcher
}

// TicketDependencies bundles repositories for ticket service.
type TicketDependencies struct {
	TicketRepo     repository.TicketRepository
	MessageRepo    repository.TicketMessageRepository
	AttachmentRepo repository.AttachmentRepository
	DepartmentRepo repository.DepartmentRepository
	TeamRepo       repository.TeamRepository
	StaffRepo      repository.StaffRepository
	HistoryRepo    repository.TicketHistoryRepository
	Dispatcher     events.Dispatcher
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
		history:     deps.HistoryRepo,
		dispatcher:  deps.Dispatcher,
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
	s.publishEvent(ctx, events.Event{
		Type:     events.EventTicketCreated,
		TicketID: ticket.ID,
		Actor:    userActor(userID),
		Payload: events.TicketCreatedPayload{
			DepartmentID: ticket.DepartmentID,
			TeamID:       ticket.TeamID,
			Priority:     ticket.Priority,
			Title:        ticket.Title,
		},
	})
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
	s.publishEvent(ctx, events.Event{
		Type:     events.EventTicketMessageAdded,
		TicketID: ticket.ID,
		Actor:    actorFromSubject(actor, actorID),
		Payload: events.TicketMessageAddedPayload{
			MessageID:   msg.ID,
			MessageType: msg.MessageType,
			AuthorType:  msg.AuthorType,
			AuthorID:    msg.AuthorID,
			BodyPreview: stringPreview(msg.Body, 120),
		},
	})
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
	oldStatus := ticket.Status
	ticket.Status = domain.TicketStatusClosed
	ticket.ClosedAt = &now
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}
	if err := s.recordStatusChange(ctx, domain.AuthorTypeUser, &userID, ticket.ID, oldStatus, ticket.Status, "user_closed"); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, events.Event{
		Type:     events.EventTicketStatusChanged,
		TicketID: ticket.ID,
		Actor:    userActor(userID),
		Payload: events.TicketStatusChangedPayload{
			OldStatus: oldStatus,
			NewStatus: ticket.Status,
			Comment:   "user_closed",
		},
	})
	return ticket, nil
}

// UpdateStatus updates ticket status by staff.
func (s *TicketService) UpdateStatus(ctx context.Context, staff *domain.StaffMember, ticketID string, newStatus domain.TicketStatus, comment string) (*domain.Ticket, error) {
	if staff == nil {
		return nil, errors.New("staff required")
	}
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if !s.staffCanAccessTicket(staff, ticket) {
		return nil, errors.New("access denied")
	}
	if !isValidTransition(ticket.Status, newStatus) {
		return nil, errors.New("invalid status transition")
	}
	oldStatus := ticket.Status
	if newStatus == domain.TicketStatusClosed {
		now := time.Now()
		ticket.ClosedAt = &now
	} else if ticket.ClosedAt != nil {
		ticket.ClosedAt = nil
	}
	ticket.Status = newStatus
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}
	if err := s.recordStatusChange(ctx, domain.AuthorTypeStaff, &staff.ID, ticket.ID, oldStatus, newStatus, comment); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, events.Event{
		Type:     events.EventTicketStatusChanged,
		TicketID: ticket.ID,
		Actor:    staffActor(staff.ID),
		Payload: events.TicketStatusChangedPayload{
			OldStatus: oldStatus,
			NewStatus: newStatus,
			Comment:   comment,
		},
	})
	return ticket, nil
}

// UpdatePriority changes ticket priority by staff.
func (s *TicketService) UpdatePriority(ctx context.Context, staff *domain.StaffMember, ticketID string, newPriority domain.TicketPriority) (*domain.Ticket, error) {
	if staff == nil {
		return nil, errors.New("staff required")
	}
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if !s.staffCanAccessTicket(staff, ticket) {
		return nil, errors.New("access denied")
	}
	oldPriority := ticket.Priority
	ticket.Priority = newPriority
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, err
	}
	if err := s.recordPriorityChange(ctx, domain.AuthorTypeStaff, &staff.ID, ticket.ID, oldPriority, newPriority); err != nil {
		return nil, err
	}
	s.publishEvent(ctx, events.Event{
		Type:     events.EventTicketPriorityChanged,
		TicketID: ticket.ID,
		Actor:    staffActor(staff.ID),
		Payload: events.TicketPriorityChangedPayload{
			OldPriority: oldPriority,
			NewPriority: newPriority,
		},
	})
	return ticket, nil
}

// ListHistoryForStaff returns history entries for staff.
func (s *TicketService) ListHistoryForStaff(ctx context.Context, staff *domain.StaffMember, ticketID string, limit, offset int) ([]domain.TicketHistory, error) {
	if s.history == nil {
		return []domain.TicketHistory{}, nil
	}
	if staff == nil {
		return nil, errors.New("staff required")
	}
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	// ensure access
	if err != nil {
		return nil, err
	}
	if !s.staffCanAccessTicket(staff, ticket) {
		return nil, errors.New("access denied")
	}
	return s.history.ListByTicket(ctx, ticketID, limit, offset)
}

// ListHistoryForUser returns user-safe history entries.
func (s *TicketService) ListHistoryForUser(ctx context.Context, userID, ticketID string) ([]domain.TicketHistory, error) {
	if s.history == nil {
		return []domain.TicketHistory{}, nil
	}
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		return nil, err
	}
	if ticket.RequesterID != userID {
		return nil, errors.New("access denied")
	}
	history, err := s.history.ListByTicket(ctx, ticketID, 100, 0)
	if err != nil {
		return nil, err
	}
	allowed := []domain.TicketHistory{}
	for _, entry := range history {
		if entry.ChangeType == domain.ChangeTypeStatus || entry.ChangeType == domain.ChangeTypeAssignee || entry.ChangeType == domain.ChangeTypeTeam {
			allowed = append(allowed, entry)
		}
	}
	return allowed, nil
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

func (s *TicketService) publishEvent(ctx context.Context, event events.Event) {
	if s.dispatcher == nil {
		return
	}
	if event.ID == "" {
		event.ID = uuid.NewString()
	}
	if event.Timestamp.IsZero() {
		event.Timestamp = time.Now()
	}
	_ = s.dispatcher.Publish(ctx, event)
}

func userActor(userID string) events.Actor {
	return events.Actor{
		Type:   domain.SubjectTypeUser,
		UserID: &userID,
	}
}

func staffActor(staffID string) events.Actor {
	return events.Actor{
		Type:    domain.SubjectTypeStaff,
		StaffID: &staffID,
	}
}

func actorFromSubject(subject domain.SubjectType, id string) events.Actor {
	switch subject {
	case domain.SubjectTypeStaff:
		return staffActor(id)
	default:
		return userActor(id)
	}
}

func stringPreview(body string, max int) string {
	body = strings.TrimSpace(body)
	if len(body) <= max {
		return body
	}
	if max <= 3 {
		return body[:max]
	}
	return body[:max-3] + "..."
}

var allowedTransitions = map[domain.TicketStatus][]domain.TicketStatus{
	domain.TicketStatusOpen:        {domain.TicketStatusInProgress, domain.TicketStatusCancelled},
	domain.TicketStatusInProgress:  {domain.TicketStatusPendingUser, domain.TicketStatusResolved, domain.TicketStatusCancelled},
	domain.TicketStatusPendingUser: {domain.TicketStatusInProgress, domain.TicketStatusResolved, domain.TicketStatusCancelled},
	domain.TicketStatusResolved:    {domain.TicketStatusClosed, domain.TicketStatusInProgress},
	domain.TicketStatusClosed:      {},
	domain.TicketStatusCancelled:   {},
}

func isValidTransition(current, next domain.TicketStatus) bool {
	for _, candidate := range allowedTransitions[current] {
		if candidate == next {
			return true
		}
	}
	return false
}

func (s *TicketService) recordStatusChange(ctx context.Context, actorType domain.MessageAuthorType, actorID *string, ticketID string, oldStatus, newStatus domain.TicketStatus, comment string) error {
	if s.history == nil {
		return nil
	}
	entry := &domain.TicketHistory{
		TicketID:      ticketID,
		ChangedByType: actorType,
		ChangedByID:   actorID,
		ChangeType:    domain.ChangeTypeStatus,
		OldValue: map[string]any{
			"status": oldStatus,
		},
		NewValue: map[string]any{
			"status":  newStatus,
			"comment": comment,
		},
	}
	return s.history.Create(ctx, entry)
}

func (s *TicketService) recordPriorityChange(ctx context.Context, actorType domain.MessageAuthorType, actorID *string, ticketID string, oldPriority, newPriority domain.TicketPriority) error {
	if s.history == nil {
		return nil
	}
	entry := &domain.TicketHistory{
		TicketID:      ticketID,
		ChangedByType: actorType,
		ChangedByID:   actorID,
		ChangeType:    domain.ChangeTypePriority,
		OldValue: map[string]any{
			"priority": oldPriority,
		},
		NewValue: map[string]any{
			"priority": newPriority,
		},
	}
	return s.history.Create(ctx, entry)
}
