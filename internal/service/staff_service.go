package service

import (
	"context"

	"github.com/jackc/pgx/v5"

	"github.com/spec-kit/ticket-service/internal/auth"
	"github.com/spec-kit/ticket-service/internal/config"
	"github.com/spec-kit/ticket-service/internal/domain"
	"github.com/spec-kit/ticket-service/internal/repository"
	apperrors "github.com/spec-kit/ticket-service/pkg/util/errorutil"
)

// StaffService manages organization entities and staff members.
type StaffService struct {
	departments repository.DepartmentRepository
	teams       repository.TeamRepository
	staff       repository.StaffRepository
	bcryptCost  int
}

// StaffListFilters define listing parameters.
type StaffListFilters struct {
	Role         *domain.StaffRole
	TeamID       *string
	DepartmentID *string
	Active       *bool
	Limit        int
	Offset       int
}

// TeamListFilters define query params for teams.
type TeamListFilters struct {
	DepartmentID    *string
	IncludeInactive bool
}

// NewStaffService constructs the service.
func NewStaffService(cfg config.Config, deps OrgDependencies) *StaffService {
	return &StaffService{
		departments: deps.DepartmentRepo,
		teams:       deps.TeamRepo,
		staff:       deps.StaffRepo,
		bcryptCost:  cfg.Auth.BcryptCost,
	}
}

// OrgDependencies encapsulates repositories required for org management.
type OrgDependencies struct {
	DepartmentRepo repository.DepartmentRepository
	TeamRepo       repository.TeamRepository
	StaffRepo      repository.StaffRepository
}

func requireAdmin(actor *domain.StaffMember) error {
	if actor == nil || actor.Role != domain.StaffRoleAdmin {
		return apperrors.NewForbidden("admin role required")
	}
	return nil
}

// CreateDepartment creates a new department.
func (s *StaffService) CreateDepartment(ctx context.Context, actor *domain.StaffMember, name, description string) (*domain.Department, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	dept := &domain.Department{
		Name:        name,
		Description: description,
		IsActive:    true,
	}
	if err := s.departments.Create(ctx, dept); err != nil {
		return nil, apperrors.MapError(err)
	}
	return dept, nil
}

// ListDepartments returns departments (optionally inactive).
func (s *StaffService) ListDepartments(ctx context.Context, actor *domain.StaffMember, includeInactive bool) ([]domain.Department, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.departments.List(ctx, includeInactive)
}

// GetDepartmentByID fetches a department.
func (s *StaffService) GetDepartmentByID(ctx context.Context, actor *domain.StaffMember, id string) (*domain.Department, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.departments.GetByID(ctx, id)
}

// UpdateDepartment modifies department metadata.
func (s *StaffService) UpdateDepartment(ctx context.Context, actor *domain.StaffMember, dept *domain.Department) (*domain.Department, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	if err := s.departments.Update(ctx, dept); err != nil {
		return nil, apperrors.MapError(err)
	}
	return dept, nil
}

// CreateTeam creates a team under a department.
func (s *StaffService) CreateTeam(ctx context.Context, actor *domain.StaffMember, departmentID, name, description string) (*domain.Team, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	dept, err := s.departments.GetByID(ctx, departmentID)
	if err != nil {
		return nil, apperrors.MapError(err)
	}
	if !dept.IsActive {
		return nil, apperrors.NewConflict("department inactive", map[string]any{"department_id": departmentID})
	}
	team := &domain.Team{
		DepartmentID: departmentID,
		Name:         name,
		Description:  description,
		IsActive:     true,
	}
	if err := s.teams.Create(ctx, team); err != nil {
		return nil, apperrors.MapError(err)
	}
	return team, nil
}

// ListTeams lists teams optionally filtered by department.
func (s *StaffService) ListTeams(ctx context.Context, actor *domain.StaffMember, filters TeamListFilters) ([]domain.Team, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.teams.List(ctx, filters.DepartmentID, filters.IncludeInactive)
}

// GetTeamByID fetches team.
func (s *StaffService) GetTeamByID(ctx context.Context, actor *domain.StaffMember, id string) (*domain.Team, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.teams.GetByID(ctx, id)
}

// UpdateTeam updates team metadata.
func (s *StaffService) UpdateTeam(ctx context.Context, actor *domain.StaffMember, team *domain.Team) (*domain.Team, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	if team.DepartmentID != "" {
		if dept, err := s.departments.GetByID(ctx, team.DepartmentID); err != nil {
			return nil, apperrors.MapError(err)
		} else if !dept.IsActive {
			return nil, apperrors.NewConflict("department inactive", map[string]any{"department_id": team.DepartmentID})
		}
	}
	if err := s.teams.Update(ctx, team); err != nil {
		return nil, apperrors.MapError(err)
	}
	return team, nil
}

// CreateStaffMember adds a new staff account.
func (s *StaffService) CreateStaffMember(ctx context.Context, actor *domain.StaffMember, name, email, password string, role domain.StaffRole, teamID *string) (*domain.StaffMember, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	if existing, err := s.staff.GetByEmail(ctx, email); err == nil && existing != nil {
		return nil, apperrors.NewConflict("staff email already exists", map[string]any{"email": email})
	} else if err != nil && err != pgx.ErrNoRows {
		return nil, apperrors.MapError(err)
	}

	var departmentID *string
	if teamID != nil && *teamID != "" {
		team, err := s.teams.GetByID(ctx, *teamID)
		if err != nil {
			return nil, apperrors.MapError(err)
		}
		if !team.IsActive {
			return nil, apperrors.NewConflict("team inactive", map[string]any{"team_id": *teamID})
		}
		departmentID = &team.DepartmentID
	}

	hash, err := auth.HashPassword(password, s.bcryptCost)
	if err != nil {
		return nil, apperrors.NewInternalError(err)
	}

	staff := &domain.StaffMember{
		Name:         name,
		Email:        email,
		PasswordHash: hash,
		Role:         role,
		DepartmentID: departmentID,
		TeamID:       teamID,
		Active:       true,
	}
	if err := s.staff.Create(ctx, staff); err != nil {
		return nil, apperrors.MapError(err)
	}
	return staff, nil
}

// ListStaffMembers lists staff with filters.
func (s *StaffService) ListStaffMembers(ctx context.Context, actor *domain.StaffMember, filters StaffListFilters) ([]domain.StaffMember, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	repoFilter := repository.StaffFilter{
		Role:         filters.Role,
		TeamID:       filters.TeamID,
		DepartmentID: filters.DepartmentID,
		Active:       filters.Active,
		Limit:        filters.Limit,
		Offset:       filters.Offset,
	}
	return s.staff.List(ctx, repoFilter)
}

// GetStaffMemberByID fetches staff.
func (s *StaffService) GetStaffMemberByID(ctx context.Context, actor *domain.StaffMember, id string) (*domain.StaffMember, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	return s.staff.GetByID(ctx, id)
}

// UpdateStaffMember updates staff details.
func (s *StaffService) UpdateStaffMember(ctx context.Context, actor *domain.StaffMember, staffID, name, email string, role domain.StaffRole, teamID *string, active bool) (*domain.StaffMember, error) {
	if err := requireAdmin(actor); err != nil {
		return nil, err
	}
	staff, err := s.staff.GetByID(ctx, staffID)
	if err != nil {
		return nil, apperrors.MapError(err)
	}
	if email != "" && email != staff.Email {
		if existing, err := s.staff.GetByEmail(ctx, email); err == nil && existing != nil && existing.ID != staff.ID {
			return nil, apperrors.NewConflict("staff email already exists", map[string]any{"email": email})
		} else if err != nil && err != pgx.ErrNoRows {
			return nil, apperrors.MapError(err)
		}
	}
	var departmentID *string
	if teamID != nil && *teamID != "" {
		team, err := s.teams.GetByID(ctx, *teamID)
		if err != nil {
			return nil, apperrors.MapError(err)
		}
		if !team.IsActive {
			return nil, apperrors.NewConflict("team inactive", map[string]any{"team_id": *teamID})
		}
		departmentID = &team.DepartmentID
	}

	staff.Name = name
	staff.Email = email
	staff.Role = role
	staff.TeamID = teamID
	staff.DepartmentID = departmentID
	staff.Active = active

	if err := s.staff.Update(ctx, staff); err != nil {
		return nil, apperrors.MapError(err)
	}
	return staff, nil
}
