package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func SetupTestDB() (repository.AuthRepository, *gorm.DB) {
	config := &gorm.Config{
		Logger: logger.Default.LogMode(logger.Silent),
	}

	db, err := gorm.Open(sqlite.Open("file::memory:?cache=shared"), config)
	if err != nil {
		panic("failed to connect database")
	}

	err = db.AutoMigrate(
		&models.Tenant{},
		&models.User{},
		&models.Role{},
		&models.RefreshToken{},
		&models.EmailVerificationToken{},
		&models.PasswordResetToken{},
	)
	if err != nil {
		panic("failed to migrate database")
	}

	return repository.NewAuthRepository(db), db
}

func TestAuthRepository_Tenants(t *testing.T) {
	repo, _ := SetupTestDB()

	t.Run("CreateAndGetTenant", func(t *testing.T) {
		tenant := &models.Tenant{
			Name:     "Test Corp",
			Slug:     "test-corp",
			Plan:     "basic",
			IsActive: true,
		}

		// Test Create
		err := repo.CreateTenant(tenant)
		assert.NoError(t, err)
		assert.NotEmpty(t, tenant.ID)

		// Test FindByID
		fetched, err := repo.FindTenantByID(tenant.ID)
		assert.NoError(t, err)
		assert.Equal(t, "Test Corp", fetched.Name)

		// Test FindBySlug
		slugFetched, err := repo.FindTenantBySlug("test-corp")
		assert.NoError(t, err)
		assert.Equal(t, tenant.ID, slugFetched.ID)
	})

	t.Run("UpdateTenant", func(t *testing.T) {
		// Création préalable
		t1 := &models.Tenant{Name: "Old Name", Slug: "old"}
		_ = repo.CreateTenant(t1)

		// Update
		updates := map[string]interface{}{"name": "New Name"}
		err := repo.UpdateTenant(t1.ID, updates)
		assert.NoError(t, err)

		// Vérification
		updated, _ := repo.FindTenantByID(t1.ID)
		assert.Equal(t, "New Name", updated.Name)
	})

	t.Run("DeleteTenant", func(t *testing.T) {
		t1 := &models.Tenant{Name: "To Delete", Slug: "del"}
		_ = repo.CreateTenant(t1)

		err := repo.DeleteTenant(t1.ID)
		assert.NoError(t, err)

		_, err = repo.FindTenantByID(t1.ID)
		assert.Error(t, err)
	})
}

func TestAuthRepository_Users(t *testing.T) {
	repo, _ := SetupTestDB()

	tenant := &models.Tenant{Name: "User Corp", Slug: "u-corp"}
	_ = repo.CreateTenant(tenant)

	t.Run("CreateAndFindUser", func(t *testing.T) {
		user := &models.User{
			Email:    "test@user.com",
			Password: "hashedpassword",
			TenantID: &tenant.ID,
		}

		// Create
		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		// FindByEmail (Scoped by Tenant)
		fetched, err := repo.FindByEmail("test@user.com", tenant.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)

		// FindUserByID
		fetchedID, err := repo.FindUserByID(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.Email, fetchedID.Email)
	})

	t.Run("MarkUserAsVerified", func(t *testing.T) {
		user := &models.User{Email: "v@v.com", TenantID: &tenant.ID, IsEmailVerified: false}
		_ = repo.CreateUser(user)

		err := repo.MarkUserAsVerified(user.ID)
		assert.NoError(t, err)

		u, _ := repo.FindUserByID(user.ID)
		assert.True(t, u.IsEmailVerified)
	})
}

func TestAuthRepository_Roles(t *testing.T) {
	repo, db := SetupTestDB()
	
	tenant := &models.Tenant{Name: "Role Corp", Slug: "r-corp"}
	_ = repo.CreateTenant(tenant)
	user := &models.User{Email: "role@u.com", TenantID: &tenant.ID}
	_ = repo.CreateUser(user)

	t.Run("RolesFlow", func(t *testing.T) {
		// 1. Create Role
		role := &models.Role{Name: "admin", Description: "Boss"}
		err := repo.CreateRole(role)
		assert.NoError(t, err)
		assert.NotEmpty(t, role.ID)

		// 2. Assign Role to User
		err = repo.AssignRoleToUser(user.ID, role.ID)
		assert.NoError(t, err)

		// 3. Get User Roles
		roles, err := repo.GetUserRoles(user.ID)
		assert.NoError(t, err)
		
		if assert.Len(t, roles, 1) {
			assert.Equal(t, "admin", roles[0].Name)
		}

		// 4. Remove Role
		err = repo.RemoveRoleFromUser(user.ID, role.ID)
		assert.NoError(t, err)

		db.Session(&gorm.Session{NewDB: true}) 
		
		rolesAfter, _ := repo.GetUserRoles(user.ID)
		assert.Len(t, rolesAfter, 0)
	})
}

func TestAuthRepository_Tokens(t *testing.T) {
	repo, _ := SetupTestDB()
	tenant := &models.Tenant{Name: "T", Slug: "t"}
	_ = repo.CreateTenant(tenant)
	user := &models.User{Email: "tok@u.com", TenantID: &tenant.ID}
	_ = repo.CreateUser(user)

	t.Run("RefreshToken", func(t *testing.T) {
		rt := &models.RefreshToken{
			UserID:    user.ID,
			Token:     "refresh-123",
			ExpiresAt: time.Now().Add(time.Hour),
			Revoked:   false,
		}
		
		err := repo.CreateRefreshToken(rt)
		assert.NoError(t, err)

		found, err := repo.GetRefreshToken("refresh-123")
		assert.NoError(t, err)
		assert.Equal(t, rt.Token, found.Token)

		err = repo.RevokeRefreshToken("refresh-123")
		assert.NoError(t, err)

		_, err = repo.GetRefreshToken("refresh-123")
		assert.Error(t, err)
	})
}