package main

import (
	_ "auth-service/docs"
	"auth-service/internal/controllers"
	"auth-service/internal/middleware"
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"auth-service/internal/seeder"
	"auth-service/internal/services"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

// @title           Auth Service API
// @version         0.0.24
// @description     Microservice d'authentification avec gestion multi-tenant et CI/CD automatisé
// @termsOfService  http://swagger.io/terms/

// @contact.name   API Support
// @contact.url    http://www.swagger.io/support

// @license.name  Apache 2.0
// @license.url   http://www.apache.org/licenses/LICENSE-2.0.html

// @BasePath  /

func main() {
	if err := godotenv.Load(); err != nil {
		log.Println("Info: Pas de fichier .env trouvé, lecture des vars système")
	}

	dsn := os.Getenv("DATABASE_URL")
	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Impossible de se connecter à la base de données:", err)
	}

	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Role{},
		&models.UserRole{},
		&models.RefreshToken{},
		&models.PasswordResetToken{},
		&models.EmailVerificationToken{},
	)
	if err != nil {
		log.Fatal("Echec de la migration DB:", err)
	}

	// Seeder: créer tenant, rôles et admin par défaut
	if err := seeder.Seed(db); err != nil {
		log.Fatal("Echec du seeder:", err)
	}

	authRepo := repository.NewAuthRepository(db)
	emailService := services.NewEmailService()

	// Services
	authService := services.NewAuthService(authRepo, emailService)
	userAdminService := services.NewUserAdminService(authRepo)
	roleService := services.NewRoleService(authRepo)
	tenantService := services.NewTenantService(authRepo)

	// Controllers
	authController := controllers.NewAuthController(authService)
	profileController := controllers.NewProfileController()
	adminController := controllers.NewAdminController(userAdminService)
	roleController := controllers.NewRoleController(roleService)
	tenantController := controllers.NewTenantController(tenantService)

	r := gin.Default()

	// Swagger / Scalar documentation
	r.GET("/api-docs/swagger.json", func(c *gin.Context) {
		c.File("/docs/swagger.json")
	})
	r.GET("/scalar", func(c *gin.Context) {
		c.Header("Content-Type", "text/html")
		c.String(200, `<!DOCTYPE html>
<html>
  <head>
    <title>Auth Service API - Scalar</title>
    <meta charset="utf-8"/>
    <meta name="viewport" content="width=device-width, initial-scale=1">
    <link href="https://fonts.googleapis.com/css?family=Montserrat:300,400,700|Roboto:300,400,700" rel="stylesheet">
    <style>
      body {
        margin: 0;
        padding: 0;
      }
    </style>
  </head>
  <body>
    <script
      id="api-reference"
      data-url="/api-docs/swagger.json"
      src="https://cdn.jsdelivr.net/npm/@scalar/api-reference"></script>
  </body>
</html>`)
	})

	// CORS origins configurables
	corsOrigins := []string{"http://localhost:3000", "http://localhost:5173"}
	if extra := os.Getenv("CORS_ORIGINS"); extra != "" {
		for _, origin := range strings.Split(extra, ",") {
			corsOrigins = append(corsOrigins, strings.TrimSpace(origin))
		}
	}

	r.Use(cors.New(cors.Config{
		AllowOrigins:     corsOrigins,
		AllowMethods:     []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Origin", "Content-Type", "Accept", "Authorization"},
		ExposeHeaders:    []string{"Content-Length"},
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	limitMiddleware := middleware.RateLimitMiddleware()

	r.GET("/health", func(c *gin.Context) {
		sqlDB, _ := db.DB()
		if err := sqlDB.Ping(); err != nil {
			c.JSON(500, gin.H{"status": "DB Dead"})
			return
		}
		c.JSON(200, gin.H{"status": "OK", "uptime": "Running"})
	})

	authGroup := r.Group("/api/auth")
	authGroup.Use(limitMiddleware)
	{
		authGroup.POST("/register", authController.Register)
		authGroup.POST("/login", authController.Login)
		authGroup.GET("/verify-email", authController.VerifyEmail)

		authGroup.POST("/refresh", authController.Refresh)
		authGroup.POST("/logout", authController.Logout)
		authGroup.POST("/forgot-password", authController.ForgotPassword)
		authGroup.POST("/reset-password", authController.ResetPassword)
	}

	protected := r.Group("/api/user")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.GET("/me", profileController.GetProfile)
	}

	// ── Endpoints internes (service-to-service, pas de JWT) ──
	internalGroup := r.Group("/internal")
	{
		internalGroup.GET("/users", func(c *gin.Context) {
			page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
			limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
			if page < 1 {
				page = 1
			}
			if limit < 1 || limit > 100 {
				limit = 10
			}
			users, total, err := userAdminService.GetAllUsers(page, limit)
			if err != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur serveur"})
				return
			}
			// Retourne les users sans le mot de passe
			type safeUser struct {
				ID              string  `json:"id"`
				Email           string  `json:"email"`
				TenantID        *string `json:"tenant_id"`
				IsEmailVerified bool    `json:"is_email_verified"`
				CreatedAt       string  `json:"created_at"`
			}
			safe := make([]safeUser, len(users))
			for i, u := range users {
				safe[i] = safeUser{
					ID:              u.ID,
					Email:           u.Email,
					TenantID:        u.TenantID,
					IsEmailVerified: u.IsEmailVerified,
					CreatedAt:       u.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
				}
			}
			c.JSON(http.StatusOK, gin.H{"data": safe, "total": total, "page": page})
		})
	}

	adminGroup := r.Group("/api/admin")
	adminGroup.Use(middleware.AuthMiddleware(), middleware.RequireRole("admin"))
	{
		// Gestion des utilisateurs
		adminGroup.POST("/promote", adminController.MakeAdmin)
		adminGroup.GET("/users", adminController.GetAllUsers)
		adminGroup.GET("/users/:id", adminController.GetUserByID)
		adminGroup.GET("/search", adminController.GetUserByEmail)

		// Gestion des rôles
		adminGroup.POST("/roles", roleController.CreateRole)
		adminGroup.GET("/roles", roleController.GetAllRoles)
		adminGroup.GET("/roles/slug/:slug", roleController.GetRoleBySlug)
		adminGroup.POST("/roles/assign", roleController.AssignRoleToUser)
		adminGroup.POST("/roles/remove", roleController.RemoveRoleFromUser)
		adminGroup.GET("/roles/user/:user_id", roleController.GetUserRoles)

		// Gestion des tenants
		adminGroup.POST("/tenants", tenantController.CreateTenant)
		adminGroup.GET("/tenants", tenantController.GetAllTenants)
		adminGroup.GET("/tenants/:id", tenantController.GetTenantByID)
		adminGroup.GET("/tenants/slug/:slug", tenantController.GetTenantBySlug)
		adminGroup.PUT("/tenants/:id", tenantController.UpdateTenant)
		adminGroup.DELETE("/tenants/:id", tenantController.DeleteTenant)
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8081"
	}
	log.Println("Service Auth PRO démarré sur le port " + port)
	r.Run(":" + port)
}