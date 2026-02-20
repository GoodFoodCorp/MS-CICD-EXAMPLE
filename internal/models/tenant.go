package models

import (
	"strings"
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type Tenant struct {
	ID        string         `gorm:"primaryKey" json:"id"`
	Name      string         `gorm:"uniqueIndex;not null" json:"name"`
	Slug      string         `gorm:"uniqueIndex;not null" json:"slug"`
	Plan      string         `gorm:"default:'free'" json:"plan"`
	IsActive  bool           `gorm:"default:true" json:"is_active"`
	Users     []User         `gorm:"foreignKey:TenantID;references:ID" json:"users,omitempty"`
	CreatedAt time.Time      `json:"created_at"`
	UpdatedAt time.Time      `json:"updated_at"`
	DeletedAt gorm.DeletedAt `gorm:"index" json:"-"`
}

func (t *Tenant) BeforeCreate(tx *gorm.DB) (err error) {
	t.ID = uuid.New().String()
	// Générer le slug automatiquement s'il n'est pas fourni
	if t.Slug == "" {
		t.Slug = GenerateSlug(t.Name)
	}
	t.Slug = strings.ToLower(t.Slug)
	return
}

// GenerateSlug crée un slug à partir d'un string
func GenerateSlug(s string) string {
	// Convertir en minuscules
	slug := strings.ToLower(s)
	// Remplacer les espaces par des tirets
	slug = strings.ReplaceAll(slug, " ", "-")
	// Supprimer les caractères spéciaux
	slug = strings.Map(func(r rune) rune {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' {
			return r
		}
		return -1
	}, slug)
	// Supprimer les tirets multiples
	for strings.Contains(slug, "--") {
		slug = strings.ReplaceAll(slug, "--", "-")
	}
	// Supprimer les tirets au début et à la fin
	slug = strings.Trim(slug, "-")
	return slug
}

type CreateTenantRequest struct {
	Name string `json:"name" binding:"required"`
	Plan string `json:"plan" binding:"required"`
}

type UpdateTenantRequest struct {
	Name     string `json:"name"`
	Plan     string `json:"plan"`
	IsActive bool   `json:"is_active"`
}
