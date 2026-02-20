package seeder

import (
	"auth-service/internal/models"
	"log"
	"os"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Seed initialise les données par défaut (tenant, roles, utilisateurs)
func Seed(db *gorm.DB) error {
	log.Println("Exécution du seeder...")

	// Créer le tenant par défaut
	tenant, err := seedDefaultTenant(db)
	if err != nil {
		return err
	}

	// Créer les rôles par défaut
	roles, err := seedDefaultRoles(db)
	if err != nil {
		return err
	}

	// Créer les utilisateurs par défaut
	if err := seedUsers(db, tenant.ID, roles); err != nil {
		return err
	}

	log.Println("Seeder terminé avec succès")
	return nil
}

func seedDefaultTenant(db *gorm.DB) (*models.Tenant, error) {
	var tenant models.Tenant
	result := db.Where("slug = ?", "default").First(&tenant)

	if result.Error == gorm.ErrRecordNotFound {
		tenant = models.Tenant{
			Name:     "Default",
			Slug:     "default",
			Plan:     "free",
			IsActive: true,
		}
		if err := db.Create(&tenant).Error; err != nil {
			return nil, err
		}
		log.Println("  ✓ Tenant 'default' créé")
	} else if result.Error != nil {
		return nil, result.Error
	} else {
		log.Println("  • Tenant 'default' existe déjà")
	}

	return &tenant, nil
}

func seedDefaultRoles(db *gorm.DB) (map[string]*models.Role, error) {
	roleDefs := []struct {
		Name        string
		Slug        string
		Description string
	}{
		{"admin", "admin", "Administrateur avec tous les droits"},
		{"manager", "manager", "Gérant de restaurant (tenant)"},
		{"user", "user", "Utilisateur standard (client)"},
		{"moderator", "moderator", "Modérateur avec droits limités"},
	}

	rolesMap := make(map[string]*models.Role)

	for _, r := range roleDefs {
		var role models.Role
		result := db.Where("slug = ?", r.Slug).First(&role)

		if result.Error == gorm.ErrRecordNotFound {
			role = models.Role{
				Name:        r.Name,
				Slug:        r.Slug,
				Description: r.Description,
			}
			if err := db.Create(&role).Error; err != nil {
				return nil, err
			}
			log.Printf("  ✓ Rôle '%s' créé (slug: %s)\n", r.Name, r.Slug)
		} else if result.Error != nil {
			return nil, result.Error
		} else {
			log.Printf("  • Rôle '%s' existe déjà\n", r.Name)
		}

		rolesMap[r.Slug] = &role
	}

	return rolesMap, nil
}

// seedUsers crée les 3 utilisateurs par défaut :
// 1. Admin (avec tenant) — gère tout
// 2. Manager (avec tenant) — gère son restaurant
// 3. Customer (sans tenant) — commande dans n'importe quel restaurant
func seedUsers(db *gorm.DB, tenantID string, roles map[string]*models.Role) error {
	type seedUser struct {
		Email    string
		Password string
		EnvEmail string
		EnvPass  string
		TenantID *string
		RoleSlug string
		Label    string
	}

	users := []seedUser{
		{
			Email:    "admin@example.com",
			Password: "Admin123!",
			EnvEmail: "ADMIN_EMAIL",
			EnvPass:  "ADMIN_PASSWORD",
			TenantID: &tenantID,
			RoleSlug: "admin",
			Label:    "Admin",
		},
		{
			Email:    "manager@example.com",
			Password: "Manager123!",
			EnvEmail: "MANAGER_EMAIL",
			EnvPass:  "MANAGER_PASSWORD",
			TenantID: &tenantID,
			RoleSlug: "manager",
			Label:    "Manager (restaurant)",
		},
		{
			Email:    "user@example.com",
			Password: "User1234!",
			EnvEmail: "USER_EMAIL",
			EnvPass:  "USER_PASSWORD",
			TenantID: nil,
			RoleSlug: "user",
			Label:    "Customer (sans tenant)",
		},
	}

	for _, su := range users {
		email := os.Getenv(su.EnvEmail)
		password := os.Getenv(su.EnvPass)
		if email == "" {
			email = su.Email
		}
		if password == "" {
			password = su.Password
		}

		var user models.User
		result := db.Where("email = ?", email).First(&user)

		if result.Error == gorm.ErrRecordNotFound {
			hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), 12)
			if err != nil {
				return err
			}

			user = models.User{
				Email:           email,
				Password:        string(hashedPassword),
				TenantID:        su.TenantID,
				IsEmailVerified: true,
			}
			if err := db.Create(&user).Error; err != nil {
				return err
			}

			// Assigner le rôle
			if role, ok := roles[su.RoleSlug]; ok {
				userRole := models.UserRole{
					UserID: user.ID,
					RoleID: role.ID,
				}
				if err := db.Create(&userRole).Error; err != nil {
					return err
				}
			}

			log.Printf("  ✓ %s '%s' créé (rôle: %s)\n", su.Label, email, su.RoleSlug)
		} else if result.Error != nil {
			return result.Error
		} else {
			log.Printf("  • %s '%s' existe déjà\n", su.Label, email)
		}
	}

	return nil
}
