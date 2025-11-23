package handlers

import (
	"net/http"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/dto"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/service"
)

// StaffTicketsHandler handles staff ticket read/message endpoints.
type StaffTicketsHandler struct {
	tickets *service.TicketService
}

// NewStaffTicketsHandler constructs handler.
func NewStaffTicketsHandler(ticketService *service.TicketService) *StaffTicketsHandler {
	return &StaffTicketsHandler{tickets: ticketService}
}

// ListStaffTickets GET /staff/tickets.
func (h *StaffTicketsHandler) ListStaffTickets(c *fiber.Ctx) error {
	staff, err := staffPrincipal(c)
	if err != nil {
		return err
	}
	filter := parseStaffTicketFilter(c)
	tickets, err := h.tickets.ListStaffTickets(c.Context(), staff, filter)
	if err != nil {
		return err
	}
	items := make([]dto.TicketSummary, 0, len(tickets))
	for i := range tickets {
		items = append(items, ticketSummary(&tickets[i]))
	}
	return c.JSON(fiber.Map{"data": items})
}

// GetStaffTicket GET /staff/tickets/:id.
func (h *StaffTicketsHandler) GetStaffTicket(c *fiber.Ctx) error {
	staff, err := staffPrincipal(c)
	if err != nil {
		return err
	}
	ticket, msgs, err := h.tickets.GetTicketForStaff(c.Context(), staff, c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": ticketDetail(ticket, msgs)})
}

// AddStaffMessage POST /staff/tickets/:id/messages.
func (h *StaffTicketsHandler) AddStaffMessage(c *fiber.Ctx) error {
	staff, err := staffPrincipal(c)
	if err != nil {
		return err
	}
	var req dto.CreateMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if strings.TrimSpace(req.Body) == "" {
		return fiber.NewError(http.StatusBadRequest, "body required")
	}
	msgType := domain.MessageTypePublicReply
	if req.MessageType != nil {
		msgType = *req.MessageType
	}
	attachments := make([]service.MessageAttachmentInput, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, service.MessageAttachmentInput{
			StorageKey: att.StorageKey,
			FileName:   att.FileName,
			MimeType:   att.MimeType,
			SizeBytes:  att.SizeBytes,
		})
	}
	msg, err := h.tickets.AddMessage(c.Context(), domain.SubjectTypeStaff, staff.ID, staff, c.Params("id"), msgType, req.Body, attachments)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": ticketMessageResponse(msg)})
}

func staffPrincipal(c *fiber.Ctx) (*domain.StaffMember, error) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.Staff == nil {
		return nil, fiber.NewError(http.StatusUnauthorized, "staff required")
	}
	return principal.Staff, nil
}

func parseStaffTicketFilter(c *fiber.Ctx) service.TicketStaffFilter {
	filter := service.TicketStaffFilter{}
	if deptID := c.Query("department_id"); deptID != "" {
		filter.DepartmentID = &deptID
	}
	if teamID := c.Query("team_id"); teamID != "" {
		filter.TeamID = &teamID
	}
	if assignee := c.Query("assignee_staff_id"); assignee != "" {
		filter.AssigneeID = &assignee
	}
	if statuses := c.Query("status"); statuses != "" {
		for _, part := range strings.Split(statuses, ",") {
			filter.Statuses = append(filter.Statuses, domain.TicketStatus(strings.TrimSpace(part)))
		}
	}
	if priorities := c.Query("priority"); priorities != "" {
		for _, part := range strings.Split(priorities, ",") {
			filter.Priorities = append(filter.Priorities, domain.TicketPriority(strings.TrimSpace(part)))
		}
	}
	if search := c.Query("search"); search != "" {
		filter.SearchTerm = &search
	}
	if createdFrom := parseTime(c.Query("created_from")); createdFrom != nil {
		filter.CreatedFrom = createdFrom
	}
	if createdTo := parseTime(c.Query("created_to")); createdTo != nil {
		filter.CreatedTo = createdTo
	}
	if updatedFrom := parseTime(c.Query("updated_from")); updatedFrom != nil {
		filter.UpdatedFrom = updatedFrom
	}
	if updatedTo := parseTime(c.Query("updated_to")); updatedTo != nil {
		filter.UpdatedTo = updatedTo
	}
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	filter.Offset = (page - 1) * pageSize
	filter.Limit = pageSize
	return filter
}
