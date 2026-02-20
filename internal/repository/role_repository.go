package repository

import "auth-service/internal/models"

func (r *authRepo) CreateRole(role *models.Role) error {
	return r.db.Create(role).Error
}

func (r *authRepo) FindRoleByName(name string) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("name = ?", name).First(&role).Error
	return &role, err
}

func (r *authRepo) FindRoleBySlug(slug string) (*models.Role, error) {
	var role models.Role
	err := r.db.Where("slug = ?", slug).First(&role).Error
	return &role, err
}

func (r *authRepo) FindRoleByID(id uint) (*models.Role, error) {
	var role models.Role
	err := r.db.First(&role, id).Error
	return &role, err
}

func (r *authRepo) GetAllRoles() ([]models.Role, error) {
	var roles []models.Role
	err := r.db.Find(&roles).Error
	return roles, err
}

func (r *authRepo) AssignRoleToUser(userID string, roleID uint) error {
	user := models.User{ID: userID}
	return r.db.Model(&user).Association("Roles").Append(&models.Role{ID: roleID})
}

func (r *authRepo) RemoveRoleFromUser(userID string, roleID uint) error {
	user := models.User{ID: userID}
	return r.db.Model(&user).Association("Roles").Delete(&models.Role{ID: roleID})
}

func (r *authRepo) GetUserRoles(userID string) ([]models.Role, error) {
	var user models.User
	if err := r.db.Preload("Roles").First(&user, "id = ?", userID).Error; err != nil {
		return nil, err
	}
	return user.Roles, nil
}
