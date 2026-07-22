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

func TestAuthRepository_Users(t *testing.T) {
	repo, _ := SetupTestDB()

	// Le restaurant vit désormais dans franchise-service : on ne référence
	// plus qu'un identifiant, sans jointure.
	restaurantID := "11111111-1111-1111-1111-111111111111"

	t.Run("CreateAndFindUser", func(t *testing.T) {
		user := &models.User{
			Email:    "test@user.com",
			Password: "hashedpassword",
			TenantID: &restaurantID,
		}

		// Create
		err := repo.CreateUser(user)
		assert.NoError(t, err)
		assert.NotEmpty(t, user.ID)

		// FindByEmail (Scoped by Tenant)
		fetched, err := repo.FindByEmail("test@user.com", restaurantID)
		assert.NoError(t, err)
		assert.Equal(t, user.ID, fetched.ID)

		// FindUserByID
		fetchedID, err := repo.FindUserByID(user.ID)
		assert.NoError(t, err)
		assert.Equal(t, user.Email, fetchedID.Email)
	})

	t.Run("MarkUserAsVerified", func(t *testing.T) {
		user := &models.User{Email: "v@v.com", TenantID: &restaurantID, IsEmailVerified: false}
		_ = repo.CreateUser(user)

		err := repo.MarkUserAsVerified(user.ID)
		assert.NoError(t, err)

		u, _ := repo.FindUserByID(user.ID)
		assert.True(t, u.IsEmailVerified)
	})
}

func TestAuthRepository_Roles(t *testing.T) {
	repo, db := SetupTestDB()

	restaurantID := "22222222-2222-2222-2222-222222222222"
	user := &models.User{Email: "role@u.com", TenantID: &restaurantID}
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
	restaurantID := "33333333-3333-3333-3333-333333333333"
	user := &models.User{Email: "tok@u.com", TenantID: &restaurantID}
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
