package http

import (
	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/api/http/handlers"
	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/domain"
)

// RouteConfig bundles dependencies for route registration.
type RouteConfig struct {
	Health         *handlers.HealthHandler
	Users          *handlers.UsersHandler
	Staff          *handlers.StaffHandler
	Tickets        *handlers.TicketsHandler
	StaffTickets   *handlers.StaffTicketsHandler
	AuthMiddleware *auth.AuthMiddleware
}

// RegisterRoutes wires HTTP routes.
func RegisterRoutes(app *fiber.App, cfg RouteConfig) {
	app.Get("/health/live", cfg.Health.Live)
	app.Get("/health/ready", cfg.Health.Ready)

	authGroup := app.Group("/auth")
	authGroup.Post("/users/register", cfg.Users.Register)
	authGroup.Post("/users/login", cfg.Users.Login)

	authGroup.Post("/staff/login", cfg.Staff.Login)
	authGroup.Post("/password/reset/request", cfg.Staff.RequestPasswordReset)
	authGroup.Post("/password/reset/confirm", cfg.Staff.ConfirmPasswordReset)

	protected := authGroup.Group("", cfg.AuthMiddleware.Handle, auth.RequireAnyRole())
	protected.Post("/password/change", cfg.Staff.ChangePassword)

	ticketsGroup := app.Group("/tickets", cfg.AuthMiddleware.Handle, auth.RequireUser())
	ticketsGroup.Post("/", cfg.Tickets.CreateTicket)
	ticketsGroup.Get("/", cfg.Tickets.ListTickets)
	ticketsGroup.Get("/:id", cfg.Tickets.GetTicket)
	ticketsGroup.Post("/:id/messages", cfg.Tickets.AddMessage)
	ticketsGroup.Post("/:id/close", cfg.Tickets.CloseTicket)

	staffBase := app.Group("/staff")
	adminGroup := staffBase.Group("", cfg.AuthMiddleware.Handle, auth.RequireStaffRole(domain.StaffRoleAdmin))
	adminGroup.Post("/departments", cfg.Staff.CreateDepartment)
	adminGroup.Get("/departments", cfg.Staff.ListDepartments)
	adminGroup.Get("/departments/:id", cfg.Staff.GetDepartment)
	adminGroup.Put("/departments/:id", cfg.Staff.UpdateDepartment)

	adminGroup.Post("/teams", cfg.Staff.CreateTeam)
	adminGroup.Get("/teams", cfg.Staff.ListTeams)
	adminGroup.Get("/teams/:id", cfg.Staff.GetTeam)
	adminGroup.Put("/teams/:id", cfg.Staff.UpdateTeam)

	adminGroup.Post("/members", cfg.Staff.CreateStaff)
	adminGroup.Get("/members", cfg.Staff.ListStaff)
	adminGroup.Get("/members/:id", cfg.Staff.GetStaff)
	adminGroup.Put("/members/:id", cfg.Staff.UpdateStaff)

	staffTickets := staffBase.Group("/tickets", cfg.AuthMiddleware.Handle, auth.RequireStaffRole(domain.StaffRoleAgent, domain.StaffRoleTeamLead, domain.StaffRoleAdmin))
	staffTickets.Get("/", cfg.StaffTickets.ListStaffTickets)
	staffTickets.Get("/:id", cfg.StaffTickets.GetStaffTicket)
	staffTickets.Post("/:id/messages", cfg.StaffTickets.AddStaffMessage)
}
