package auth

import (
	"net/http"
	"strings"

	"github.com/gofiber/fiber/v2"
	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/repository"
)

const principalKey = "auth_principal"

// Principal represents the authenticated caller.
type Principal struct {
	SubjectType domain.SubjectType
	User        *domain.User
	Staff       *domain.StaffMember
	Role        *domain.StaffRole
}

// AuthMiddleware validates bearer tokens and loads principals.
type AuthMiddleware struct {
	tokens *TokenManager
	users  repository.UserRepository
	staff  repository.StaffRepository
}

// NewAuthMiddleware constructs middleware.
func NewAuthMiddleware(tokens *TokenManager, users repository.UserRepository, staff repository.StaffRepository) *AuthMiddleware {
	return &AuthMiddleware{tokens: tokens, users: users, staff: staff}
}

// Handle enforces authentication for protected routes.
func (m *AuthMiddleware) Handle(c *fiber.Ctx) error {
	authHeader := c.Get("Authorization")
	if authHeader == "" {
		return fiber.NewError(http.StatusUnauthorized, "missing authorization header")
	}

	parts := strings.SplitN(authHeader, " ", 2)
	if len(parts) != 2 || !strings.EqualFold(parts[0], "Bearer") {
		return fiber.NewError(http.StatusUnauthorized, "invalid authorization header")
	}

	claims, err := m.tokens.ParseToken(parts[1])
	if err != nil {
		return fiber.NewError(http.StatusUnauthorized, "invalid token")
	}

	principal := &Principal{SubjectType: claims.Subject, Role: claims.Role}

	switch claims.Subject {
	case domain.SubjectTypeUser:
		user, err := m.users.GetByID(c.Context(), claims.SubjectID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fiber.NewError(http.StatusUnauthorized, "user not found")
			}
			return err
		}
		principal.User = user
	case domain.SubjectTypeStaff:
		staff, err := m.staff.GetByID(c.Context(), claims.SubjectID)
		if err != nil {
			if err == pgx.ErrNoRows {
				return fiber.NewError(http.StatusUnauthorized, "staff not found")
			}
			return err
		}
		principal.Staff = staff
	default:
		return fiber.NewError(http.StatusUnauthorized, "unknown subject")
	}

	c.Locals(principalKey, principal)
	return c.Next()
}

// PrincipalFromContext retrieves the authenticated entity.
func PrincipalFromContext(c *fiber.Ctx) (*Principal, bool) {
	val := c.Locals(principalKey)
	if val == nil {
		return nil, false
	}
	principal, ok := val.(*Principal)
	return principal, ok
}
