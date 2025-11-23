package handlers

import (
	"net/http"
	"strconv"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/dto"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/service"
)

// StaffHandler exposes staff/auth endpoints.
type StaffHandler struct {
	authService *service.AuthService
	orgService  *service.StaffService
}

// NewStaffHandler constructs handler.
func NewStaffHandler(authService *service.AuthService, orgService *service.StaffService) *StaffHandler {
	return &StaffHandler{authService: authService, orgService: orgService}
}

// Login handles POST /auth/staff/login.
func (h *StaffHandler) Login(c *fiber.Ctx) error {
	var req dto.StaffLoginRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" || req.Password == "" {
		return fiber.NewError(http.StatusBadRequest, "email and password required")
	}

	staff, token, exp, err := h.authService.LoginStaff(c.Context(), req.Email, req.Password)
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, err.Error())
	}

	return c.JSON(fiber.Map{
		"data": fiber.Map{
			"staff": staffResponse(staff),
			"auth":  dto.AuthResponse{Token: token, ExpiresAt: exp},
		},
	})
}

// RequestPasswordReset handles POST /auth/password/reset/request.
func (h *StaffHandler) RequestPasswordReset(c *fiber.Ctx) error {
	var req dto.PasswordResetRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Email == "" {
		return fiber.NewError(http.StatusBadRequest, "email required")
	}

	token, err := h.authService.RequestPasswordReset(c.Context(), req.Email)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusAccepted).JSON(fiber.Map{
		"data": fiber.Map{
			"reset_token": token.Token,
			"expires_at":  token.ExpiresAt,
		},
	})
}

// ConfirmPasswordReset handles POST /auth/password/reset/confirm.
func (h *StaffHandler) ConfirmPasswordReset(c *fiber.Ctx) error {
	var req dto.PasswordResetConfirmRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Token == "" || req.NewPassword == "" {
		return fiber.NewError(http.StatusBadRequest, "token and new password required")
	}

	if err := h.authService.ConfirmPasswordReset(c.Context(), req.Token, req.NewPassword); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"status": "password_reset"}})
}

// ChangePassword handles POST /auth/password/change.
func (h *StaffHandler) ChangePassword(c *fiber.Ctx) error {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok {
		return fiber.NewError(http.StatusUnauthorized, "authentication required")
	}

	var req dto.PasswordChangeRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.CurrentPassword == "" || req.NewPassword == "" {
		return fiber.NewError(http.StatusBadRequest, "current and new password required")
	}

	subject := service.AuthSubject{Type: principal.SubjectType}
	switch principal.SubjectType {
	case domain.SubjectTypeUser:
		subject.ID = principal.User.ID
	case domain.SubjectTypeStaff:
		subject.ID = principal.Staff.ID
	default:
		return fiber.NewError(http.StatusUnauthorized, "unknown subject")
	}

	if err := h.authService.ChangePassword(c.Context(), subject, req.CurrentPassword, req.NewPassword); err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": fiber.Map{"status": "password_changed"}})
}

// CreateDepartment handles POST /staff/departments.
func (h *StaffHandler) CreateDepartment(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	var req dto.DepartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Name == "" {
		return fiber.NewError(http.StatusBadRequest, "name required")
	}
	dept, err := h.orgService.CreateDepartment(c.Context(), admin, req.Name, req.Description)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": departmentResponse(dept)})
}

// ListDepartments handles GET /staff/departments.
func (h *StaffHandler) ListDepartments(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	includeInactive := parseBoolQuery(c, "include_inactive", false)
	depts, err := h.orgService.ListDepartments(c.Context(), admin, includeInactive)
	if err != nil {
		return err
	}
	resp := make([]dto.DepartmentResponse, 0, len(depts))
	for i := range depts {
		resp = append(resp, departmentResponse(&depts[i]))
	}
	return c.JSON(fiber.Map{"data": resp})
}

// GetDepartment handles GET /staff/departments/:id.
func (h *StaffHandler) GetDepartment(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	dept, err := h.orgService.GetDepartmentByID(c.Context(), admin, c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": departmentResponse(dept)})
}

// UpdateDepartment handles PUT /staff/departments/:id.
func (h *StaffHandler) UpdateDepartment(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	dept, err := h.orgService.GetDepartmentByID(c.Context(), admin, c.Params("id"))
	if err != nil {
		return err
	}
	var req dto.DepartmentRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Name != "" {
		dept.Name = req.Name
	}
	dept.Description = req.Description
	if req.IsActive != nil {
		dept.IsActive = *req.IsActive
	}
	updated, err := h.orgService.UpdateDepartment(c.Context(), admin, dept)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": departmentResponse(updated)})
}

// CreateTeam handles POST /staff/teams.
func (h *StaffHandler) CreateTeam(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	var req dto.TeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.DepartmentID == "" || req.Name == "" {
		return fiber.NewError(http.StatusBadRequest, "department_id and name required")
	}
	team, err := h.orgService.CreateTeam(c.Context(), admin, req.DepartmentID, req.Name, req.Description)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": teamResponse(team)})
}

// ListTeams handles GET /staff/teams.
func (h *StaffHandler) ListTeams(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	var deptID *string
	if val := c.Query("department_id"); val != "" {
		deptID = &val
	}
	includeInactive := parseBoolQuery(c, "include_inactive", false)
	teams, err := h.orgService.ListTeams(c.Context(), admin, service.TeamListFilters{
		DepartmentID:    deptID,
		IncludeInactive: includeInactive,
	})
	if err != nil {
		return err
	}
	resp := make([]dto.TeamResponse, 0, len(teams))
	for i := range teams {
		resp = append(resp, teamResponse(&teams[i]))
	}
	return c.JSON(fiber.Map{"data": resp})
}

// GetTeam handles GET /staff/teams/:id.
func (h *StaffHandler) GetTeam(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	team, err := h.orgService.GetTeamByID(c.Context(), admin, c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": teamResponse(team)})
}

// UpdateTeam handles PUT /staff/teams/:id.
func (h *StaffHandler) UpdateTeam(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	team, err := h.orgService.GetTeamByID(c.Context(), admin, c.Params("id"))
	if err != nil {
		return err
	}
	var req dto.TeamRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Name != "" {
		team.Name = req.Name
	}
	if req.Description != "" {
		team.Description = req.Description
	}
	if req.DepartmentID != "" {
		team.DepartmentID = req.DepartmentID
	}
	if req.IsActive != nil {
		team.IsActive = *req.IsActive
	}
	updated, err := h.orgService.UpdateTeam(c.Context(), admin, team)
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": teamResponse(updated)})
}

// CreateStaff handles POST /staff/members.
func (h *StaffHandler) CreateStaff(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	var req dto.StaffCreateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	if req.Name == "" || req.Email == "" || req.Password == "" {
		return fiber.NewError(http.StatusBadRequest, "name, email, password required")
	}
	staff, err := h.orgService.CreateStaffMember(c.Context(), admin, req.Name, req.Email, req.Password, req.Role, req.TeamID)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.Status(http.StatusCreated).JSON(fiber.Map{"data": staffResponse(staff)})
}

// ListStaff handles GET /staff/members.
func (h *StaffHandler) ListStaff(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	filters := parseStaffListFilters(c)
	list, err := h.orgService.ListStaffMembers(c.Context(), admin, filters)
	if err != nil {
		return err
	}
	resp := make([]dto.StaffResponse, 0, len(list))
	for i := range list {
		resp = append(resp, staffResponse(&list[i]))
	}
	return c.JSON(fiber.Map{"data": resp})
}

// GetStaff handles GET /staff/members/:id.
func (h *StaffHandler) GetStaff(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	staff, err := h.orgService.GetStaffMemberByID(c.Context(), admin, c.Params("id"))
	if err != nil {
		return err
	}
	return c.JSON(fiber.Map{"data": staffResponse(staff)})
}

// UpdateStaff handles PUT /staff/members/:id.
func (h *StaffHandler) UpdateStaff(c *fiber.Ctx) error {
	admin, err := h.requireAdminPrincipal(c)
	if err != nil {
		return err
	}
	var req dto.StaffUpdateRequest
	if err := c.BodyParser(&req); err != nil {
		return fiber.NewError(http.StatusBadRequest, "invalid payload")
	}
	updated, err := h.orgService.UpdateStaffMember(c.Context(), admin, c.Params("id"), req.Name, req.Email, req.Role, req.TeamID, req.Active)
	if err != nil {
		return fiber.NewError(http.StatusBadRequest, err.Error())
	}
	return c.JSON(fiber.Map{"data": staffResponse(updated)})
}

func (h *StaffHandler) requireAdminPrincipal(c *fiber.Ctx) (*domain.StaffMember, error) {
	principal, ok := auth.PrincipalFromContext(c)
	if !ok || principal.Staff == nil {
		return nil, fiber.NewError(http.StatusUnauthorized, "staff required")
	}
	if principal.Staff.Role != domain.StaffRoleAdmin {
		return nil, fiber.NewError(http.StatusForbidden, "admin role required")
	}
	return principal.Staff, nil
}

func parseBoolQuery(c *fiber.Ctx, key string, defaultVal bool) bool {
	if val := c.Query(key); val != "" {
		if parsed, err := strconv.ParseBool(val); err == nil {
			return parsed
		}
	}
	return defaultVal
}

func parseStaffListFilters(c *fiber.Ctx) service.StaffListFilters {
	var filters service.StaffListFilters
	if roleStr := c.Query("role"); roleStr != "" {
		role := domain.StaffRole(roleStr)
		filters.Role = &role
	}
	if teamID := c.Query("team_id"); teamID != "" {
		filters.TeamID = &teamID
	}
	if deptID := c.Query("department_id"); deptID != "" {
		filters.DepartmentID = &deptID
	}
	if active := c.Query("active"); active != "" {
		if val, err := strconv.ParseBool(active); err == nil {
			filters.Active = &val
		}
	}
	page := parseIntQuery(c, "page", 1)
	pageSize := parseIntQuery(c, "page_size", 50)
	filters.Offset = (page - 1) * pageSize
	filters.Limit = pageSize
	return filters
}

func parseIntQuery(c *fiber.Ctx, key string, defaultVal int) int {
	if val := c.Query(key); val != "" {
		if parsed, err := strconv.Atoi(val); err == nil && parsed > 0 {
			return parsed
		}
	}
	return defaultVal
}

func departmentResponse(dept *domain.Department) dto.DepartmentResponse {
	return dto.DepartmentResponse{
		ID:          dept.ID,
		Name:        dept.Name,
		Description: dept.Description,
		IsActive:    dept.IsActive,
	}
}

func teamResponse(team *domain.Team) dto.TeamResponse {
	return dto.TeamResponse{
		ID:           team.ID,
		DepartmentID: team.DepartmentID,
		Name:         team.Name,
		Description:  team.Description,
		IsActive:     team.IsActive,
	}
}

func staffResponse(staff *domain.StaffMember) dto.StaffResponse {
	return dto.StaffResponse{
		ID:           staff.ID,
		Name:         staff.Name,
		Email:        staff.Email,
		Role:         staff.Role,
		DepartmentID: staff.DepartmentID,
		TeamID:       staff.TeamID,
		Active:       staff.Active,
	}
}
