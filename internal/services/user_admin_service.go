package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
)

type UserAdminService interface {
	GetAllUsers(page int, limit int) ([]models.User, int64, error)
	GetUserByID(userID string) (*models.User, error)
	GetUserByEmail(email string) (*models.User, error)
	PromoteUserToAdmin(userID string) error
}

type userAdminService struct {
	repo repository.AuthRepository
}

func NewUserAdminService(repo repository.AuthRepository) UserAdminService {
	return &userAdminService{repo: repo}
}

func (s *userAdminService) GetAllUsers(p, l int) ([]models.User, int64, error) {
	return s.repo.FindAll(p, l)
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
