package service

import (
	"context"
	"sort"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/events"
	"github.com/spec-kit/ticket-service/internal/repository"
	apperrors "github.com/spec-kit/ticket-service/pkg/util/errorutil"
)

// AssignmentService handles ticket assignment operations.
type AssignmentService struct {
	tickets     repository.TicketRepository
	staff       repository.StaffRepository
	teams       repository.TeamRepository
	historyRepo repository.TicketHistoryRepository
	dispatcher  events.Dispatcher
}

// AssignmentDependencies bundles repositories.
type AssignmentDependencies struct {
	TicketRepo  repository.TicketRepository
	StaffRepo   repository.StaffRepository
	TeamRepo    repository.TeamRepository
	HistoryRepo repository.TicketHistoryRepository
	Dispatcher  events.Dispatcher
}

// NewAssignmentService creates the service.
func NewAssignmentService(deps AssignmentDependencies) *AssignmentService {
	return &AssignmentService{
		tickets:     deps.TicketRepo,
		staff:       deps.StaffRepo,
		teams:       deps.TeamRepo,
		historyRepo: deps.HistoryRepo,
		dispatcher:  deps.Dispatcher,
	}
}

// SelfAssignTicket allows a staff member to assign ticket to themselves.
func (s *AssignmentService) SelfAssignTicket(ctx context.Context, staff *domain.StaffMember, ticketID string) (*domain.Ticket, error) {
	if staff == nil {
		return nil, apperrors.NewUnauthorized("staff required")
	}
	if staff.Role != domain.StaffRoleAgent && staff.Role != domain.StaffRoleTeamLead && staff.Role != domain.StaffRoleAdmin {
		return nil, apperrors.NewForbidden("insufficient role for self assign")
	}

	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("ticket", map[string]any{"ticket_id": ticketID})
		}
		return nil, apperrors.MapError(err)
	}
	if !s.staffCanAccess(staff, ticket) {
		return nil, apperrors.NewForbidden("access denied")
	}
	oldAssignee := ticket.AssigneeID
	ticket.AssigneeID = &staff.ID
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, apperrors.MapError(err)
	}
	if err := s.recordAssigneeChange(ctx, staff.ID, ticket.ID, oldAssignee, ticket.AssigneeID); err != nil {
		return nil, apperrors.MapError(err)
	}
	s.publishAssignmentEvent(ctx, staff.ID, events.TicketAssignedPayload{
		AssigneeStaffID: ticket.AssigneeID,
		TeamID:          ticket.TeamID,
	}, ticket.ID)
	return ticket, nil
}

// AssignTicketToStaff assigns ticket to provided staff (TEAM_LEAD/ADMIN).
func (s *AssignmentService) AssignTicketToStaff(ctx context.Context, actor *domain.StaffMember, ticketID, assigneeStaffID string) (*domain.Ticket, error) {
	if err := requireAssignPriv(actor); err != nil {
		return nil, apperrors.MapError(err)
	}
	assignee, err := s.staff.GetByID(ctx, assigneeStaffID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("staff", map[string]any{"staff_id": assigneeStaffID})
		}
		return nil, apperrors.MapError(err)
	}
	if !assignee.Active {
		return nil, apperrors.NewConflict("assignee inactive", map[string]any{"staff_id": assigneeStaffID})
	}

	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("ticket", map[string]any{"ticket_id": ticketID})
		}
		return nil, apperrors.MapError(err)
	}
	if !s.staffCanAccess(actor, ticket) {
		return nil, apperrors.NewForbidden("access denied")
	}
	if !s.staffMatchesTicketScope(assignee, ticket) && actor.Role != domain.StaffRoleAdmin {
		return nil, apperrors.NewForbidden("assignee outside ticket scope")
	}
	oldAssignee := ticket.AssigneeID
	ticket.AssigneeID = &assignee.ID
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, apperrors.MapError(err)
	}
	if err := s.recordAssigneeChange(ctx, actor.ID, ticket.ID, oldAssignee, ticket.AssigneeID); err != nil {
		return nil, apperrors.MapError(err)
	}
	s.publishAssignmentEvent(ctx, actor.ID, events.TicketAssignedPayload{
		AssigneeStaffID: ticket.AssigneeID,
		TeamID:          ticket.TeamID,
	}, ticket.ID)
	return ticket, nil
}

// AssignTicketToTeam reassigns ticket to another team (TEAM_LEAD/ADMIN).
func (s *AssignmentService) AssignTicketToTeam(ctx context.Context, actor *domain.StaffMember, ticketID, teamID string) (*domain.Ticket, error) {
	if err := requireAssignPriv(actor); err != nil {
		return nil, err
	}
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("team", map[string]any{"team_id": teamID})
		}
		return nil, apperrors.MapError(err)
	}
	if !team.IsActive {
		return nil, apperrors.NewConflict("team inactive", map[string]any{"team_id": teamID})
	}
	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("ticket", map[string]any{"ticket_id": ticketID})
		}
		return nil, apperrors.MapError(err)
	}
	if !s.staffCanAccess(actor, ticket) {
		return nil, apperrors.NewForbidden("access denied")
	}
	oldTeam := ticket.TeamID
	oldDept := ticket.DepartmentID
	ticket.TeamID = &team.ID
	ticket.DepartmentID = team.DepartmentID
	ticket.AssigneeID = nil
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, apperrors.MapError(err)
	}
	if err := s.recordTeamChange(ctx, actor.ID, ticket.ID, oldTeam, ticket.TeamID); err != nil {
		return nil, apperrors.MapError(err)
	}
	if oldDept != team.DepartmentID {
		if err := s.recordDepartmentChange(ctx, actor.ID, ticket.ID, oldDept, ticket.DepartmentID); err != nil {
			return nil, apperrors.MapError(err)
		}
	}
	s.publishAssignmentEvent(ctx, actor.ID, events.TicketAssignedPayload{
		AssigneeStaffID: nil,
		TeamID:          ticket.TeamID,
	}, ticket.ID)
	return ticket, nil
}

// AutoAssignTicket selects an assignee for ticket.
func (s *AssignmentService) AutoAssignTicket(ctx context.Context, ticketID, teamID string) (*domain.Ticket, error) {
	team, err := s.teams.GetByID(ctx, teamID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("team", map[string]any{"team_id": teamID})
		}
		return nil, apperrors.MapError(err)
	}
	if !team.IsActive {
		return nil, apperrors.NewConflict("team inactive", map[string]any{"team_id": teamID})
	}
	filter := repository.StaffFilter{
		TeamID: &teamID,
		Active: ptrBool(true),
		Limit:  1000,
	}
	staffList, err := s.staff.List(ctx, filter)
	if err != nil {
		return nil, apperrors.MapError(err)
	}
	if len(staffList) == 0 {
		return nil, apperrors.NewConflict("no eligible staff for team", map[string]any{"team_id": teamID})
	}
	sort.Slice(staffList, func(i, j int) bool {
		return staffList[i].CreatedAt.Before(staffList[j].CreatedAt)
	})

	ticket, err := s.tickets.GetByID(ctx, ticketID)
	if err != nil {
		if errors.Is(err, pgx.ErrNoRows) {
			return nil, apperrors.NewNotFound("ticket", map[string]any{"ticket_id": ticketID})
		}
		return nil, apperrors.MapError(err)
	}
	index := selectIndex(ticket.ID, len(staffList))
	assignee := staffList[index]
	oldAssignee := ticket.AssigneeID
	oldTeam := ticket.TeamID
	oldDept := ticket.DepartmentID
	ticket.TeamID = &team.ID
	ticket.DepartmentID = team.DepartmentID
	ticket.AssigneeID = &assignee.ID
	if err := s.tickets.Update(ctx, ticket); err != nil {
		return nil, apperrors.MapError(err)
	}
	if err := s.recordTeamChange(ctx, assignee.ID, ticket.ID, oldTeam, ticket.TeamID); err != nil {
		return nil, apperrors.MapError(err)
	}
	if oldDept != team.DepartmentID {
		if err := s.recordDepartmentChange(ctx, assignee.ID, ticket.ID, oldDept, ticket.DepartmentID); err != nil {
			return nil, apperrors.MapError(err)
		}
	}
	if err := s.recordAssigneeChange(ctx, assignee.ID, ticket.ID, oldAssignee, ticket.AssigneeID); err != nil {
		return nil, apperrors.MapError(err)
	}
	s.publishAssignmentEvent(ctx, assignee.ID, events.TicketAssignedPayload{
		AssigneeStaffID: ticket.AssigneeID,
		TeamID:          ticket.TeamID,
	}, ticket.ID)
	return ticket, nil
}

func selectIndex(key string, length int) int {
	if length == 0 {
		return 0
	}
	sum := 0
	for _, ch := range key {
		sum += int(ch)
	}
	return sum % length
}

func ptrBool(v bool) *bool {
	return &v
}

func requireAssignPriv(staff *domain.StaffMember) error {
	if staff == nil {
		return apperrors.NewUnauthorized("staff required")
	}
	if staff.Role != domain.StaffRoleTeamLead && staff.Role != domain.StaffRoleAdmin {
		return apperrors.NewForbidden("insufficient role for assignment")
	}
	return nil
}

func (s *AssignmentService) staffCanAccess(staff *domain.StaffMember, ticket *domain.Ticket) bool {
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

func (s *AssignmentService) staffMatchesTicketScope(staff *domain.StaffMember, ticket *domain.Ticket) bool {
	if staff == nil {
		return false
	}
	if staff.TeamID != nil && ticket.TeamID != nil && *staff.TeamID == *ticket.TeamID {
		return true
	}
	if staff.DepartmentID != nil && *staff.DepartmentID == ticket.DepartmentID {
		return true
	}
	return false
}

func (s *AssignmentService) recordAssigneeChange(ctx context.Context, actorID string, ticketID string, oldAssignee, newAssignee *string) error {
	return s.historyRepo.Create(ctx, &domain.TicketHistory{
		TicketID:      ticketID,
		ChangedByType: domain.AuthorTypeStaff,
		ChangedByID:   &actorID,
		ChangeType:    domain.ChangeTypeAssignee,
		OldValue: map[string]any{
			"assignee_staff_id": oldAssignee,
		},
		NewValue: map[string]any{
			"assignee_staff_id": newAssignee,
		},
	})
}

func (s *AssignmentService) recordTeamChange(ctx context.Context, actorID string, ticketID string, oldTeam, newTeam *string) error {
	return s.historyRepo.Create(ctx, &domain.TicketHistory{
		TicketID:      ticketID,
		ChangedByType: domain.AuthorTypeStaff,
		ChangedByID:   &actorID,
		ChangeType:    domain.ChangeTypeTeam,
		OldValue: map[string]any{
			"team_id": oldTeam,
		},
		NewValue: map[string]any{
			"team_id": newTeam,
		},
	})
}

func (s *AssignmentService) recordDepartmentChange(ctx context.Context, actorID string, ticketID string, oldDept, newDept string) error {
	return s.historyRepo.Create(ctx, &domain.TicketHistory{
		TicketID:      ticketID,
		ChangedByType: domain.AuthorTypeStaff,
		ChangedByID:   &actorID,
		ChangeType:    domain.ChangeTypeDepartment,
		OldValue: map[string]any{
			"department_id": oldDept,
		},
		NewValue: map[string]any{
			"department_id": newDept,
		},
	})
}

func (s *AssignmentService) publishAssignmentEvent(ctx context.Context, actorID string, payload events.TicketAssignedPayload, ticketID string) {
	if s.dispatcher == nil {
		return
	}
	event := events.Event{
		ID:        uuid.NewString(),
		Type:      events.EventTicketAssigned,
		TicketID:  ticketID,
		Actor:     events.Actor{Type: domain.SubjectTypeStaff, StaffID: &actorID},
		Timestamp: time.Now(),
		Payload:   payload,
	}
	_ = s.dispatcher.Publish(ctx, event)
}
