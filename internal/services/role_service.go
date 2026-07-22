package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"errors"
)

type RoleService interface {
	CreateRole(name, description string) error
	AssignRoleToUser(userID string, roleID uint) error
	RemoveRoleFromUser(userID string, roleID uint) error
	GetUserRoles(userID string) ([]models.Role, error)
	GetAllRoles() ([]models.Role, error)
	GetRoleBySlug(slug string) (*models.Role, error)
}

type roleService struct {
	repo repository.AuthRepository
}

func NewRoleService(repo repository.AuthRepository) RoleService {
	return &roleService{repo: repo}
}

func (s *roleService) CreateRole(name, description string) error {
	// Vérifie que le rôle n'existe pas déjà
	_, err := s.repo.FindRoleByName(name)
	if err == nil {
		return errors.New("ce rôle existe déjà")
	}

	role := &models.Role{
		Name:        name,
		Description: description,
	}
	return s.repo.CreateRole(role)
}

func (s *roleService) AssignRoleToUser(userID string, roleID uint) error {
	_, err := s.repo.FindUserByID(userID)
	if err != nil {
		return errors.New("utilisateur introuvable")
	}

	_, err = s.repo.FindRoleByID(roleID)
	if err != nil {
		return errors.New("rôle introuvable")
	}

	return s.repo.AssignRoleToUser(userID, roleID)
}

func (s *roleService) RemoveRoleFromUser(userID string, roleID uint) error {
	return s.repo.RemoveRoleFromUser(userID, roleID)
}

func (s *roleService) GetUserRoles(userID string) ([]models.Role, error) {
	return s.repo.GetUserRoles(userID)
}

func (s *roleService) GetAllRoles() ([]models.Role, error) {
	return s.repo.GetAllRoles()
}

func (s *roleService) GetRoleBySlug(slug string) (*models.Role, error) {
	return s.repo.FindRoleBySlug(slug)
}
