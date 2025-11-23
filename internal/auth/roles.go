package auth

import (
	"net/http"

	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/domain"
)

// RequireUser ensures an END_USER is authenticated.
func RequireUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal, ok := PrincipalFromContext(c)
		if !ok || principal.SubjectType != domain.SubjectTypeUser {
			return fiber.NewError(http.StatusForbidden, "end-user required")
		}
		return c.Next()
	}
}

// RequireStaffRole ensures the staff principal has one of the allowed roles.
func RequireStaffRole(allowed ...domain.StaffRole) fiber.Handler {
	allowedSet := make(map[domain.StaffRole]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}

	return func(c *fiber.Ctx) error {
		principal, ok := PrincipalFromContext(c)
		if !ok || principal.SubjectType != domain.SubjectTypeStaff || principal.Staff == nil {
			return fiber.NewError(http.StatusForbidden, "staff role required")
		}
		if len(allowedSet) == 0 {
			return c.Next()
		}
		if _, exists := allowedSet[principal.Staff.Role]; !exists {
			return fiber.NewError(http.StatusForbidden, "insufficient role")
		}
		return c.Next()
	}
}

// RequireAnyRole ensures caller is authenticated (user or staff).
func RequireAnyRole() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := PrincipalFromContext(c); !ok {
			return fiber.NewError(http.StatusUnauthorized, http.StatusText(http.StatusUnauthorized))
		}
		return c.Next()
	}
}
