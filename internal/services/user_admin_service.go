package services

import (
	"errors"
	"strings"

	"auth-service/internal/models"
	"auth-service/internal/repository"

	"github.com/google/uuid"
)

// Staff management errors, mapped to 400 / 403 / 404 by the controller.
var (
	ErrUnknownRole      = errors.New("rôle inconnu")
	ErrManagerNeedsShop = errors.New("un gérant doit être rattaché à un restaurant")
	ErrSelfDemotion     = errors.New("vous ne pouvez pas retirer vos propres droits d'administrateur")
	ErrUserNotFound     = errors.New("utilisateur introuvable")
	maxUsersPerPage     = 100
	assignableRoleSlugs = map[string]bool{"user": true, "manager": true, "livreur": true, "admin": true}
)

type UserAdminService interface {
	// GetAllUsers lists users newest first; search filters on the email.
	GetAllUsers(page int, limit int, search string) ([]models.User, int64, error)
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	PromoteUserToAdmin(userID string) error
	// SetUserAccess gives a user one role and, for a manager, a restaurant.
	// actorID is the admin doing it: they cannot demote themselves.
	SetUserAccess(actorID, userID, roleSlug, restaurantID string) (*models.User, error)
}

type userAdminService struct {
	repo repository.AuthRepository
}

func NewUserAdminService(repo repository.AuthRepository) UserAdminService {
	return &userAdminService{repo: repo}
}

func (s *userAdminService) GetAllUsers(p, l int, search string) ([]models.User, int64, error) {
	if p < 1 {
		p = 1
	}
	if l < 1 || l > maxUsersPerPage {
		l = maxUsersPerPage
	}
	return s.repo.FindAll(p, l, search)
}

func (s *userAdminService) SetUserAccess(actorID, userID, roleSlug, restaurantID string) (*models.User, error) {
	roleSlug = strings.TrimSpace(roleSlug)
	if !assignableRoleSlugs[roleSlug] {
		return nil, ErrUnknownRole
	}
	if actorID == userID && roleSlug != "admin" {
		return nil, ErrSelfDemotion
	}
	var tenantID *string
	if roleSlug == "manager" {
		if _, err := uuid.Parse(strings.TrimSpace(restaurantID)); err != nil {
			return nil, ErrManagerNeedsShop
		}
		id := strings.TrimSpace(restaurantID)
		tenantID = &id
	}
	user, err := s.repo.FindUserByID(userID)
	if err != nil {
		return nil, ErrUserNotFound
	}
	role, err := s.repo.FindRoleBySlug(roleSlug)
	if err != nil {
		return nil, ErrUnknownRole
	}
	if err := s.repo.SetUserAccess(userID, role.ID, tenantID); err != nil {
		return nil, err
	}
	user.TenantID = tenantID
	user.Roles = []models.Role{*role}
	return user, nil
}

func (s *userAdminService) GetUserByID(id string) (*models.User, error) {
	return s.repo.FindUserByID(id)
}

func (s *userAdminService) GetUserByEmail(e string) (*models.User, error) {
	return s.repo.FindGlobalByEmail(e)
}

func (s *userAdminService) PromoteUserToAdmin(userID string) error {
	_, err := s.repo.FindUserByID(userID)
	if err != nil {
		return err
	}
	adminRole, err := s.repo.FindRoleByName("admin")
	if err != nil {
		// Crée le rôle s'il n'existe pas
		adminRole = &models.Role{
			Name:        "admin",
			Description: "Administrateur",
		}
		if err := s.repo.CreateRole(adminRole); err != nil {
			return err
		}
		adminRole, _ = s.repo.FindRoleByName("admin")
	}
	return s.repo.AssignRoleToUser(userID, adminRole.ID)
}
