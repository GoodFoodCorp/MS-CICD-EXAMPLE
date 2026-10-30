package repository

import (
	"auth-service/internal/models"

	"gorm.io/gorm"
)

type AuthRepository interface {

	// Users
	CreateUser(user *models.User) error
	FindByEmail(email string, tenantID string) (*models.User, error)
	FindUserByID(userID string) (*models.User, error)
	FindGlobalByEmail(email string) (*models.User, error)
	// FindAll lists users with their roles, newest first; search filters on
	// the email (case-insensitive substring).
	FindAll(page int, limit int, search string) ([]models.User, int64, error)
	DeleteUser(userID string) error

	// Tokens
	CreateRefreshToken(token *models.RefreshToken) error
	GetRefreshToken(token string) (*models.RefreshToken, error)
	RevokeRefreshToken(token string) error
	RevokeAllRefreshTokensForUser(userID string) error
	CreatePasswordResetToken(token *models.PasswordResetToken) error
	GetPasswordResetToken(token string) (*models.PasswordResetToken, error)
	MarkPasswordResetTokenAsUsed(token string) error
	UpdateUserPassword(userID string, newPassword string) error
	CreateEmailVerificationToken(token *models.EmailVerificationToken) error
	FindEmailVerificationToken(token string) (*models.EmailVerificationToken, error)
	DeleteEmailVerificationToken(id uint) error
	MarkUserAsVerified(userID string) error

	// OAuth
	FindOAuthAccount(provider, providerUserID string) (*models.OAuthAccount, error)
	CreateOAuthAccount(account *models.OAuthAccount) error

	// Roles
	CreateRole(role *models.Role) error
	FindRoleByName(name string) (*models.Role, error)
	FindRoleBySlug(slug string) (*models.Role, error)
	FindRoleByID(id uint) (*models.Role, error)
	GetAllRoles() ([]models.Role, error)
	AssignRoleToUser(userID string, roleID uint) error
	RemoveRoleFromUser(userID string, roleID uint) error
	GetUserRoles(userID string) ([]models.Role, error)
	// SetUserAccess makes roleID the user's only role and tenantID their
	// restaurant, and revokes their sessions so the change applies at once.
	SetUserAccess(userID string, roleID uint, tenantID *string) error
}

type authRepo struct {
	db *gorm.DB
}

func (r *authRepo) DeleteUser(userID string) error {
	return r.db.Transaction(func(tx *gorm.DB) error {
		if err := tx.Where("user_id = ?", userID).Delete(&models.UserRole{}).Error; err != nil { return err }
		if err := tx.Where("user_id = ?", userID).Delete(&models.RefreshToken{}).Error; err != nil { return err }
		if err := tx.Where("id = ?", userID).Delete(&models.User{}).Error; err != nil { return err }
		return nil
	})
}

func NewAuthRepository(db *gorm.DB) AuthRepository {
	return &authRepo{db: db}
}
