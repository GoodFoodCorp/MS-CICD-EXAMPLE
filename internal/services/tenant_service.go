package services

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
)

type TenantService interface {
	CreateTenant(req *models.CreateTenantRequest) (*models.Tenant, error)
	GetTenantByID(id string) (*models.Tenant, error)
	GetTenantBySlug(slug string) (*models.Tenant, error)
	GetAllTenants(page int, limit int) ([]models.Tenant, int64, error)
	UpdateTenant(id string, req *models.UpdateTenantRequest) error
	DeleteTenant(id string) error
}

type tenantService struct {
	repo repository.AuthRepository
}

func NewTenantService(repo repository.AuthRepository) TenantService {
	return &tenantService{repo: repo}
}

func (s *tenantService) CreateTenant(req *models.CreateTenantRequest) (*models.Tenant, error) {
	slug := models.GenerateSlug(req.Name)

	tenant := &models.Tenant{
		Name:     req.Name,
		Slug:     slug,
		Plan:     req.Plan,
		IsActive: true,
	}

	if err := s.repo.CreateTenant(tenant); err != nil {
		return nil, err
	}

	return tenant, nil
}

func (s *tenantService) GetTenantByID(id string) (*models.Tenant, error) {
	return s.repo.FindTenantByID(id)
}

func (s *tenantService) GetTenantBySlug(slug string) (*models.Tenant, error) {
	return s.repo.FindTenantBySlug(models.GenerateSlug(slug))
}

func (s *tenantService) GetAllTenants(page int, limit int) ([]models.Tenant, int64, error) {
	return s.repo.GetAllTenants(page, limit)
}

func (s *tenantService) UpdateTenant(id string, req *models.UpdateTenantRequest) error {
	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Plan != "" {
		updates["plan"] = req.Plan
	}
	updates["is_active"] = req.IsActive

	return s.repo.UpdateTenant(id, updates)
}

func (s *tenantService) DeleteTenant(id string) error {
	return s.repo.DeleteTenant(id)
}
