package auth

import (
	"github.com/gofiber/fiber/v2"

	"github.com/spec-kit/ticket-service/internal/domain"
	apperrors "github.com/spec-kit/ticket-service/pkg/util/errorutil"
)

// RequireUser ensures an END_USER is authenticated.
func RequireUser() fiber.Handler {
	return func(c *fiber.Ctx) error {
		principal, ok := PrincipalFromContext(c)
		if !ok || principal.SubjectType != domain.SubjectTypeUser {
			return apperrors.NewForbidden("end-user required")
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
			return apperrors.NewForbidden("staff role required")
		}
		if len(allowedSet) == 0 {
			return c.Next()
		}
		if _, exists := allowedSet[principal.Staff.Role]; !exists {
			return apperrors.NewForbidden("insufficient role")
		}
		return c.Next()
	}
}

// RequireAnyRole ensures caller is authenticated (user or staff).
func RequireAnyRole() fiber.Handler {
	return func(c *fiber.Ctx) error {
		if _, ok := PrincipalFromContext(c); !ok {
			return apperrors.NewUnauthorized("authentication required")
		}
		return c.Next()
	}
}
