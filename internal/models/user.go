package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type User struct {
	ID string `gorm:"primaryKey" json:"id"`
	// Référence au restaurant (franchise-service) — pas de jointure : chaque
	// service possède ses propres données.
	TenantID        *string        `gorm:"index" json:"tenant_id"`
	Email           string         `gorm:"uniqueIndex;not null" json:"email"`
	Password        string         `gorm:"not null" json:"-"`
	IsEmailVerified bool           `gorm:"default:false" json:"is_email_verified"`
	Roles           []Role         `gorm:"many2many:user_roles;" json:"roles"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
	DeletedAt       gorm.DeletedAt `gorm:"index" json:"-"`
}

func (u *User) BeforeCreate(tx *gorm.DB) (err error) {
	u.ID = uuid.New().String()
	return
}

type RegisterRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required,min=8"`
	TenantID string `json:"tenantID"`
}

type LoginRequest struct {
	Email    string `json:"email" binding:"required,email"`
	Password string `json:"password" binding:"required"`
}

type LogoutRequest struct {
}
