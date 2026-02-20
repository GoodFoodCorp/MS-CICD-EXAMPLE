package models

import (
	"strings"
	"time"

	"gorm.io/gorm"
)

type Role struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"uniqueIndex;not null" json:"name"`
	Slug        string    `gorm:"uniqueIndex;not null" json:"slug"`
	Description string    `json:"description"`
	Users       []User    `gorm:"many2many:user_roles;" json:"-"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

// BeforeCreate génère automatiquement le slug à partir du nom
func (r *Role) BeforeCreate(tx *gorm.DB) (err error) {
	if r.Slug == "" {
		r.Slug = GenerateSlug(r.Name)
	}
	r.Slug = strings.ToLower(r.Slug)
	return
}

type UserRole struct {
	ID        uint      `gorm:"primaryKey"`
	UserID    string    `gorm:"index;not null"`
	RoleID    uint      `gorm:"index;not null"`
	CreatedAt time.Time
}
