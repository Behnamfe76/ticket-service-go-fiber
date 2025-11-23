package service

import (
	"context"
	"errors"
	"time"

	"github.com/google/uuid"
	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/config"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/repository"
)

// AuthSubject identifies the caller when changing password.
type AuthSubject struct {
	Type domain.SubjectType
	ID   string
}

// AuthService coordinates registration and login flows.
type AuthService struct {
	users      repository.UserRepository
	staff      repository.StaffRepository
	resets     repository.PasswordResetRepository
	tokenMgr   *auth.TokenManager
	bcryptCost int
	resetTTL   time.Duration
}

// AuthDependencies encapsulates repo requirements for auth service.
type AuthDependencies struct {
	UserRepo          repository.UserRepository
	StaffRepo         repository.StaffRepository
	PasswordResetRepo repository.PasswordResetRepository
}

// NewAuthService builds the service.
func NewAuthService(cfg config.Config, deps AuthDependencies) *AuthService {
	return &AuthService{
		users:      deps.UserRepo,
		staff:      deps.StaffRepo,
		resets:     deps.PasswordResetRepo,
		tokenMgr:   auth.NewTokenManager(cfg.Auth.JWTSecret, cfg.Auth.AccessTokenTTLMinutes),
		bcryptCost: cfg.Auth.BcryptCost,
		resetTTL:   time.Duration(cfg.Auth.PasswordResetTTLMinutes) * time.Minute,
	}
}

// RegisterUser creates a new end-user account.
func (s *AuthService) RegisterUser(ctx context.Context, name, email, password string) (*domain.User, string, time.Time, error) {
	if _, err := s.users.GetByEmail(ctx, email); err == nil {
		return nil, "", time.Time{}, errors.New("email already registered")
	} else if err != nil && err != pgx.ErrNoRows {
		return nil, "", time.Time{}, err
	}

	hash, err := auth.HashPassword(password, s.bcryptCost)
	if err != nil {
		return nil, "", time.Time{}, err
	}

	user := &domain.User{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Status:       domain.UserStatusActive,
	}
	if err := s.users.Create(ctx, user); err != nil {
		return nil, "", time.Time{}, err
	}

	token, exp, err := s.tokenMgr.GenerateToken(user.ID, domain.SubjectTypeUser, nil)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return user, token, exp, nil
}

// LoginUser authenticates an end-user.
func (s *AuthService) LoginUser(ctx context.Context, email, password string) (*domain.User, string, time.Time, error) {
	user, err := s.users.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if err := auth.ComparePassword(user.PasswordHash, password); err != nil {
		return nil, "", time.Time{}, errors.New("invalid credentials")
	}
	token, exp, err := s.tokenMgr.GenerateToken(user.ID, domain.SubjectTypeUser, nil)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return user, token, exp, nil
}

// LoginStaff authenticates staff and returns role-bearing token.
func (s *AuthService) LoginStaff(ctx context.Context, email, password string) (*domain.StaffMember, string, time.Time, error) {
	staff, err := s.staff.GetByEmail(ctx, email)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	if !staff.Active {
		return nil, "", time.Time{}, errors.New("staff inactive")
	}
	if err := auth.ComparePassword(staff.PasswordHash, password); err != nil {
		return nil, "", time.Time{}, errors.New("invalid credentials")
	}
	token, exp, err := s.tokenMgr.GenerateToken(staff.ID, domain.SubjectTypeStaff, &staff.Role)
	if err != nil {
		return nil, "", time.Time{}, err
	}
	return staff, token, exp, nil
}

// Logout currently no-ops for stateless JWT approach.
func (s *AuthService) Logout(_ context.Context, _ string) error {
	return nil
}

// RequestPasswordReset persists a reset token for either user or staff email.
func (s *AuthService) RequestPasswordReset(ctx context.Context, email string) (*repository.PasswordResetToken, error) {
	subjectType := domain.SubjectTypeUser
	subjectID := ""

	if user, err := s.users.GetByEmail(ctx, email); err == nil {
		subjectID = user.ID
	} else if err == pgx.ErrNoRows {
		staff, staffErr := s.staff.GetByEmail(ctx, email)
		if staffErr != nil {
			return nil, staffErr
		}
		subjectType = domain.SubjectTypeStaff
		subjectID = staff.ID
	} else {
		return nil, err
	}

	token := &repository.PasswordResetToken{
		SubjectType: string(subjectType),
		SubjectID:   subjectID,
		Token:       uuid.NewString(),
		ExpiresAt:   time.Now().Add(s.resetTTL),
	}
	if err := s.resets.Create(ctx, token); err != nil {
		return nil, err
	}
	return token, nil
}

// ConfirmPasswordReset validates the reset token and updates password.
func (s *AuthService) ConfirmPasswordReset(ctx context.Context, tokenStr, newPassword string) error {
	token, err := s.resets.GetByToken(ctx, tokenStr)
	if err != nil {
		return err
	}
	if token.UsedAt != nil || time.Now().After(token.ExpiresAt) {
		return errors.New("token expired or used")
	}

	hash, err := auth.HashPassword(newPassword, s.bcryptCost)
	if err != nil {
		return err
	}

	switch domain.SubjectType(token.SubjectType) {
	case domain.SubjectTypeUser:
		user, err := s.users.GetByID(ctx, token.SubjectID)
		if err != nil {
			return err
		}
		user.PasswordHash = hash
		if err := s.users.Update(ctx, user); err != nil {
			return err
		}
	case domain.SubjectTypeStaff:
		staff, err := s.staff.GetByID(ctx, token.SubjectID)
		if err != nil {
			return err
		}
		staff.PasswordHash = hash
		if err := s.staff.Update(ctx, staff); err != nil {
			return err
		}
	default:
		return errors.New("unknown subject type")
	}

	return s.resets.MarkUsed(ctx, token.ID)
}

// ChangePassword verifies current password before updating to new hash.
func (s *AuthService) ChangePassword(ctx context.Context, subject AuthSubject, currentPassword, newPassword string) error {
	hash, err := auth.HashPassword(newPassword, s.bcryptCost)
	if err != nil {
		return err
	}

	switch subject.Type {
	case domain.SubjectTypeUser:
		user, err := s.users.GetByID(ctx, subject.ID)
		if err != nil {
			return err
		}
		if err := auth.ComparePassword(user.PasswordHash, currentPassword); err != nil {
			return errors.New("invalid credentials")
		}
		user.PasswordHash = hash
		return s.users.Update(ctx, user)
	case domain.SubjectTypeStaff:
		staff, err := s.staff.GetByID(ctx, subject.ID)
		if err != nil {
			return err
		}
		if err := auth.ComparePassword(staff.PasswordHash, currentPassword); err != nil {
			return errors.New("invalid credentials")
		}
		staff.PasswordHash = hash
		return s.staff.Update(ctx, staff)
	default:
		return errors.New("unknown subject")
	}
}

// TokenManager exposes the underlying token manager for middleware usage.
func (s *AuthService) TokenManager() *auth.TokenManager {
	return s.tokenMgr
}
