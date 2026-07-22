package repository

import "auth-service/internal/models"

func (r *authRepo) CreateUser(user *models.User) error {
	return r.db.Create(user).Error
}

func (r *authRepo) FindAll(page int, limit int) ([]models.User, int64, error) {
	var users []models.User
	var total int64
	offset := (page - 1) * limit
	if err := r.db.Model(&models.User{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	err := r.db.Offset(offset).Limit(limit).Find(&users).Error
	return users, total, err
}

func (r *authRepo) FindByEmail(email string, tenantID string) (*models.User, error) {
	var user models.User
	query := r.db.Where("email = ?", email)
	if tenantID != "" {
		query = query.Where("tenant_id = ?", tenantID)
	} else {
		query = query.Where("tenant_id IS NULL")
	}
	err := query.First(&user).Error
	return &user, err
}

func (r *authRepo) FindUserByID(userID string) (*models.User, error) {
	var user models.User
	err := r.db.First(&user, "id = ?", userID).Error
	return &user, err
}

func (r *authRepo) FindGlobalByEmail(email string) (*models.User, error) {
	var user models.User
	err := r.db.Where("email = ?", email).First(&user).Error
	return &user, err

}
