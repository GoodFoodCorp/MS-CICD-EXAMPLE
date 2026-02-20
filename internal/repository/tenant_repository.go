package repository

import "auth-service/internal/models"

func (r *authRepo) CreateTenant(tenant *models.Tenant) error {
	return r.db.Create(tenant).Error
}

func (r *authRepo) FindTenantByID(id string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.First(&tenant, "id = ?", id).Error
	return &tenant, err
}

func (r *authRepo) FindTenantBySlug(slug string) (*models.Tenant, error) {
	var tenant models.Tenant
	err := r.db.First(&tenant, "slug = ?", slug).Error
	return &tenant, err
}

func (r *authRepo) GetAllTenants(page int, limit int) ([]models.Tenant, int64, error) {
	var tenants []models.Tenant
	var total int64
	offset := (page - 1) * limit
	if err := r.db.Model(&models.Tenant{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Offset(offset).Limit(limit).Find(&tenants).Error
	return tenants, total, err
}

func (r *authRepo) UpdateTenant(id string, updates map[string]interface{}) error {
	return r.db.Model(&models.Tenant{}).Where("id = ?", id).Updates(updates).Error
}

func (r *authRepo) DeleteTenant(id string) error {
	return r.db.Where("id = ?", id).Delete(&models.Tenant{}).Error
}
