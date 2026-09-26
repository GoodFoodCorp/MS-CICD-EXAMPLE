package models

import "time"

// OAuthAccount lie un compte externe (Google, Microsoft…) à un utilisateur
// local. Un même utilisateur peut en lier plusieurs.
type OAuthAccount struct {
	ID             uint      `gorm:"primaryKey"`
	UserID         string    `gorm:"index;not null"`
	Provider       string    `gorm:"index:idx_provider_subject,unique;not null"`
	ProviderUserID string    `gorm:"index:idx_provider_subject,unique;not null"`
	Email          string    `gorm:"not null"`
	CreatedAt      time.Time
}

// OAuthProfile est le profil normalisé renvoyé par un fournisseur, une fois le
// code d'autorisation échangé.
type OAuthProfile struct {
	Provider      string
	Subject       string // identifiant stable chez le fournisseur
	Email         string
	EmailVerified bool
	Name          string
}

// OAuthProviderInfo décrit un fournisseur activé, pour que le front n'affiche
// que les boutons réellement utilisables.
type OAuthProviderInfo struct {
	Provider string `json:"provider"`
	Label    string `json:"label"`
}
