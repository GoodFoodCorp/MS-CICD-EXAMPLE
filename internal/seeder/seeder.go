package seeder

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"log"
	"os"
	"time"

	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

// Seed initialise les données par défaut (restaurants/tenants, rôles, utilisateurs)
func Seed(db *gorm.DB) error {
	log.Println("Exécution du seeder...")

	// Les restaurants (franchises) appartiennent désormais à franchise-service :
	// on récupère leurs identifiants pour y rattacher les managers.
	restaurants := fetchRestaurants()

	// Créer les rôles par défaut
	roles, err := seedDefaultRoles(db)
	if err != nil {
		return err
	}

	// Créer les utilisateurs par défaut
	if err := seedUsers(db, restaurants, roles); err != nil {
		return err
	}

	log.Println("Seeder terminé avec succès")
	return nil
}

// fetchRestaurants interroge franchise-service (propriétaire des restaurants)
// et indexe les restaurants par slug. Best-effort avec quelques essais : si le
// service n'est pas encore prêt, les managers ne seront pas rattachés à cette
// exécution — un redémarrage rattrape le coup.
func fetchRestaurants() map[string]services.Restaurant {
	for attempt := 1; attempt <= 5; attempt++ {
		bySlug, err := services.FetchRestaurantsBySlug()
		if err == nil {
			log.Printf("  • %d restaurant(s) récupéré(s) depuis franchise-service\n", len(bySlug))
			return bySlug
		}
		log.Printf("  [RETRY] franchise-service injoignable (%d/5): %v\n", attempt, err)
		time.Sleep(3 * time.Second)
	}
	log.Println("  [WARNING] franchise-service injoignable : les managers ne seront pas rattachés à un restaurant")
	return map[string]services.Restaurant{}
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
		{"livreur", "livreur", "Livreur (coursier, non lié à un tenant)"},
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

// seedUsers crée les utilisateurs par défaut :
//  1. Admin (siège) — gère tout
//  2. Manager République — gère le restaurant Good Food République
//  3. Manager Montparnasse — gère le restaurant Good Food Montparnasse
//  4. Customer (sans tenant) — commande dans n'importe quel restaurant
//  5. Livreur (sans tenant)
func seedUsers(db *gorm.DB, restaurants map[string]services.Restaurant, roles map[string]*models.Role) error {
	// Un pointeur nil laisse l'utilisateur sans restaurant (client, livreur, ou
	// manager si franchise-service n'a pas répondu).
	restaurantID := func(slug string) *string {
		if r, ok := restaurants[slug]; ok {
			id := r.ID
			return &id
		}
		return nil
	}
	republique := restaurantID("default")
	montparnasse := restaurantID("montparnasse")

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
			TenantID: republique,
			RoleSlug: "admin",
			Label:    "Admin (siège)",
		},
		{
			Email:    "manager@example.com",
			Password: "Manager123!",
			EnvEmail: "MANAGER_EMAIL",
			EnvPass:  "MANAGER_PASSWORD",
			TenantID: republique,
			RoleSlug: "manager",
			Label:    "Manager (République)",
		},
		{
			Email:    "manager2@example.com",
			Password: "Manager123!",
			EnvEmail: "MANAGER2_EMAIL",
			EnvPass:  "MANAGER2_PASSWORD",
			TenantID: montparnasse,
			RoleSlug: "manager",
			Label:    "Manager (Montparnasse)",
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
		{
			Email:    "livreur@example.com",
			Password: "Livreur123!",
			EnvEmail: "LIVREUR_EMAIL",
			EnvPass:  "LIVREUR_PASSWORD",
			TenantID: nil,
			RoleSlug: "livreur",
			Label:    "Livreur (sans tenant)",
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
