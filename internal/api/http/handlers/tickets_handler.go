package handlers

import (
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/dto"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/service"
)

// TicketsHandler manages end-user ticket endpoints.
type TicketsHandler struct {
	service *service.TicketService
}

// NewTicketsHandler constructs handler.
func NewTicketsHandler(ticketService *service.TicketService) *TicketsHandler {
	return &TicketsHandler{service: ticketService}
}

// CreateTicket POST /tickets.
func (h *TicketsHandler) CreateTicket(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.User == nil {
		return fiber.NewError(http.StatusUnauthorized, "user required")
	}
	var req dto.CreateTicketRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.DepartmentID == "" || req.Title == "" || req.Description == "" {
		return fiber.NewError(http.StatusBadRequest, "department_id, title, description required")
	}

	input := service.TicketCreateInput{
		DepartmentID: req.DepartmentID,
		TeamID:       req.TeamID,
		Title:        req.Title,
		Description:  req.Description,
		Priority:     req.Priority,
		Tags:         req.Tags,
	}
	ticket, err := h.service.CreateTicket(c.Context(), principal.User.ID, input)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": ticketSummary(ticket)})
}

// ListTickets GET /tickets.
func (h *TicketsHandler) ListTickets(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.User == nil {
		return fiber.NewError(http.StatusUnauthorized, "user required")
	}
	filter := parseUserTicketQuery(c)
	tickets, err := h.service.ListUserTickets(c.Context(), principal.User.ID, filter)
	if err != nil {
		return err
	}
	items := make([]dto.TicketSummary, 0, len(tickets))
	for i := range tickets {
		items = append(items, ticketSummary(&tickets[i]))
	}
	return c.JSON(fiber.Map{"data": items})
}

// GetTicket GET /tickets/:id.
func (h *TicketsHandler) GetTicket(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.User == nil {
		return fiber.NewError(http.StatusUnauthorized, "user required")
	}
	ticket, msgs, err := h.service.GetTicketForUser(c.Context(), principal.User.ID, c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": ticketDetail(ticket, msgs)})
}

// AddMessage POST /tickets/:id/messages.
func (h *TicketsHandler) AddMessage(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.User == nil {
		return fiber.NewError(http.StatusUnauthorized, "user required")
	}
	var req dto.CreateMessageRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if strings.TrimSpace(req.Body) == "" {
		return fiber.NewError(http.StatusBadRequest, "body required")
	}
	messageType := domain.MessageTypePublicReply
	attachments := make([]service.MessageAttachmentInput, 0, len(req.Attachments))
	for _, att := range req.Attachments {
		attachments = append(attachments, service.MessageAttachmentInput{
			StorageKey: att.StorageKey,
			FileName:   att.FileName,
			MimeType:   att.MimeType,
			SizeBytes:  att.SizeBytes,
		})
	}
	msg, err := h.service.AddMessage(c.Context(), domain.SubjectTypeUser, principal.User.ID, nil, c.Params("id"), messageType, req.Body, attachments)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": ticketMessageResponse(msg)})
}

// CloseTicket POST /tickets/:id/close.
func (h *TicketsHandler) CloseTicket(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.User == nil {
		return fiber.NewError(http.StatusUnauthorized, "user required")
	}
	ticket, err := h.service.CloseTicketAsUser(c.Context(), principal.User.ID, c.Params("id"))
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": ticketSummary(ticket)})
}

func parseUserTicketQuery(c *fiber.Ctx) service.TicketUserFilter {
	filter := service.TicketUserFilter{}
	if statusStr := c.Query("status"); statusStr != "" {
		for _, part := range strings.Split(statusStr, ",") {
			filter.Statuses = append(filter.Statuses, domain.TicketStatus(strings.TrimSpace(part)))
		}
	}
	if priorityStr := c.Query("priority"); priorityStr != "" {
		for _, part := range strings.Split(priorityStr, ",") {
			filter.Priorities = append(filter.Priorities, domain.TicketPriority(strings.TrimSpace(part)))
		}
	}
	if from := parseTime(c.Query("created_from")); from != nil {
		filter.CreatedFrom = from
	}
	if to := parseTime(c.Query("created_to")); to != nil {
		filter.CreatedTo = to
	}
	page := parseInt(c.Query("page"), 1)
	pageSize := parseInt(c.Query("page_size"), 20)
	filter.Offset = (page - 1) * pageSize
	filter.Limit = pageSize
	return filter
}

func parseTime(val string) *time.Time {
	if val == "" {
		return nil
	}
	t, err := time.Parse(time.RFC3339, val)
	if err != nil {
		return nil
	}
	return &t
}

func parseInt(val string, def int) int {
	if val == "" {
		return def
	}
	parsed, err := strconv.Atoi(val)
	if err != nil || parsed <= 0 {
		return def
	}
	return parsed
}

func ticketSummary(ticket *domain.Ticket) dto.TicketSummary {
	return dto.TicketSummary{
		ID:           ticket.ID,
		ExternalKey:  ticket.ExternalKey,
		DepartmentID: ticket.DepartmentID,
		TeamID:       ticket.TeamID,
		Title:        ticket.Title,
		Status:       ticket.Status,
		Priority:     ticket.Priority,
		Tags:         ticket.Tags,
		CreatedAt:    ticket.CreatedAt,
		UpdatedAt:    ticket.UpdatedAt,
	}
}

func ticketDetail(ticket *domain.Ticket, messages []domain.TicketMessage) dto.TicketDetailResponse {
	msgs := make([]dto.TicketMessageResponse, 0, len(messages))
	for i := range messages {
		msgs = append(msgs, ticketMessageResponse(&messages[i]))
	}
	return dto.TicketDetailResponse{
		ID:           ticket.ID,
		ExternalKey:  ticket.ExternalKey,
		DepartmentID: ticket.DepartmentID,
		TeamID:       ticket.TeamID,
		Title:        ticket.Title,
		Description:  ticket.Description,
		Status:       ticket.Status,
		Priority:     ticket.Priority,
		Tags:         ticket.Tags,
		CreatedAt:    ticket.CreatedAt,
		UpdatedAt:    ticket.UpdatedAt,
		ClosedAt:     ticket.ClosedAt,
		Messages:     msgs,
	}
}

func ticketMessageResponse(msg *domain.TicketMessage) dto.TicketMessageResponse {
	attachments := make([]dto.AttachmentResponse, 0, len(msg.Attachments))
	for _, att := range msg.Attachments {
		attachments = append(attachments, dto.AttachmentResponse{
			ID:        att.ID,
			FileName:  att.FileName,
			MimeType:  att.MimeType,
			SizeBytes: att.SizeBytes,
		})
	}
	return dto.TicketMessageResponse{
		ID:          msg.ID,
		MessageType: msg.MessageType,
		AuthorType:  msg.AuthorType,
		AuthorID:    msg.AuthorID,
		Body:        msg.Body,
		Attachments: attachments,
		CreatedAt:   msg.CreatedAt,
	}
}
