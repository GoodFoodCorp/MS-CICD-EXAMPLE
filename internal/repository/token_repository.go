package repository

import (
	"auth-service/internal/models"
	"time"
)

// ─── Refresh Tokens ─────────────────────────────────

func (r *authRepo) CreateRefreshToken(token *models.RefreshToken) error {
	return r.db.Create(token).Error
}

func (r *authRepo) GetRefreshToken(token string) (*models.RefreshToken, error) {
	var rt models.RefreshToken
	err := r.db.Where("token = ? AND revoked = ? AND expires_at > ?", token, false, time.Now()).First(&rt).Error
	return &rt, err
}

func (r *authRepo) RevokeRefreshToken(token string) error {
	return r.db.Model(&models.RefreshToken{}).Where("token = ?", token).Update("revoked", true).Error
}

// ─── Password Reset Tokens ─────────────────────────

func (r *authRepo) CreatePasswordResetToken(token *models.PasswordResetToken) error {
	return r.db.Create(token).Error
}

func (r *authRepo) GetPasswordResetToken(token string) (*models.PasswordResetToken, error) {
	var prt models.PasswordResetToken
	err := r.db.Where("token = ? AND used = ? AND expires_at > ?", token, false, time.Now()).First(&prt).Error
	return &prt, err
}

func (r *authRepo) MarkPasswordResetTokenAsUsed(token string) error {
	return r.db.Model(&models.PasswordResetToken{}).Where("token = ?", token).Update("used", true).Error
}

func (r *authRepo) UpdateUserPassword(userID string, newPassword string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("password", newPassword).Error
}

// ─── Email Verification Tokens ──────────────────────

func (r *authRepo) CreateEmailVerificationToken(token *models.EmailVerificationToken) error {
	return r.db.Create(token).Error
}

func (r *authRepo) FindEmailVerificationToken(token string) (*models.EmailVerificationToken, error) {
	var evt models.EmailVerificationToken
	err := r.db.Where("token = ? AND expires_at > ?", token, time.Now()).First(&evt).Error
	return &evt, err
}

func (r *authRepo) DeleteEmailVerificationToken(id uint) error {
	return r.db.Delete(&models.EmailVerificationToken{}, id).Error
}

func (r *authRepo) MarkUserAsVerified(userID string) error {
	return r.db.Model(&models.User{}).Where("id = ?", userID).Update("is_email_verified", true).Error
}
