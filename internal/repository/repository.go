package repository

import (
	"auth-service/internal/models"

	"gorm.io/gorm"
)

type AuthRepository interface {
	// Tenants
	CreateTenant(tenant *models.Tenant) error
	FindTenantByID(id string) (*models.Tenant, error)
	FindTenantBySlug(slug string) (*models.Tenant, error)
	GetAllTenants(page int, limit int) ([]models.Tenant, int64, error)
	UpdateTenant(id string, updates map[string]interface{}) error
	DeleteTenant(id string) error

	// Users
	CreateUser(user *models.User) error
	FindByEmail(email string, tenantID string) (*models.User, error)
	FindUserByID(userID string) (*models.User, error)
	FindGlobalByEmail(email string) (*models.User, error)
	FindAll(page int, limit int) ([]models.User, int64, error)

	// Tokens
	CreateRefreshToken(token *models.RefreshToken) error
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token string) error
	CreatePasswordResetToken(token *models.PasswordResetToken) error
	GetPasswordResetToken(token string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(token string) error
	UpdateUserPassword(userID string, newPassword string) error
	CreateEmailVerificationToken(token *models.EmailVerificationToken) error
	FindEmailVerificationToken(token string) (*models.EmailVerificationToken, error)
	DeleteEmailVerificationToken(id uint) error
	MarkUserAsVerified(userID string) error

	// Roles
	CreateRole(role *models.Role) error
	FindRoleByName(name string) (*models.Role, error)
	FindRoleBySlug(slug string) (*models.Role, error)
	FindRoleByID(id uint) (*models.Role, error)
	GetAllRoles() ([]models.Role, error)
	AssignRoleToUser(userID string, roleID uint) error
	RemoveRoleFromUser(userID string, roleID uint) error
	GetUserRoles(userID string) ([]models.Role, error)
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepo{db: db}
}
