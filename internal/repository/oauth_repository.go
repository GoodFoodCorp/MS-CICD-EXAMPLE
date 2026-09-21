package repository

import "auth-service/internal/models"

func (r *authRepo) FindOAuthAccount(provider, providerUserID string) (*models.OAuthAccount, error) {
	var account models.OAuthAccount
	err := r.db.Where("provider = ? AND provider_user_id = ?", provider, providerUserID).First(&account).Error
	return &account, err
}

func (r *authRepo) CreateOAuthAccount(account *models.OAuthAccount) error {
	return r.db.Create(account).Error
}
