package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/repository"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gorm.io/driver/sqlite"
	"gorm.io/gorm"
	"gorm.io/gorm/logger"
)

func erasureDB(t *testing.T) (repository.AuthRepository, *gorm.DB) {
	t.Helper()
	db, err := gorm.Open(sqlite.Open("file:erasure?mode=memory&cache=private"), &gorm.Config{Logger: logger.Default.LogMode(logger.Silent)})
	require.NoError(t, err)
	require.NoError(t, db.AutoMigrate(&models.User{}, &models.Role{}, &models.UserRole{}, &models.RefreshToken{},
		&models.EmailVerificationToken{}, &models.PasswordResetToken{}, &models.OAuthAccount{}))
	return repository.NewAuthRepository(db), db
}

func userWithTraces(t *testing.T, repo repository.AuthRepository, db *gorm.DB, email string) string {
	t.Helper()
	u := &models.User{Email: email, Password: "hash"}
	require.NoError(t, repo.CreateUser(u))
	require.NoError(t, db.Create(&models.RefreshToken{UserID: u.ID, Token: "h-" + email, ExpiresAt: time.Now().Add(time.Hour)}).Error)
	require.NoError(t, db.Create(&models.PasswordResetToken{UserID: u.ID, Token: "r-" + email, ExpiresAt: time.Now()}).Error)
	return u.ID
}

func count(db *gorm.DB, model any) int64 {
	var n int64
	db.Unscoped().Model(model).Count(&n)
	return n
}

// Right to erasure: nothing of the account stays, and the address can sign up again.
func TestDeleteUser_ErasesForGood(t *testing.T) {
	repo, db := erasureDB(t)
	id := userWithTraces(t, repo, db, "camille@example.com")

	require.NoError(t, repo.DeleteUser(id))

	assert.Zero(t, count(db, &models.User{}), "no soft-deleted row keeping the email and hash")
	assert.Zero(t, count(db, &models.RefreshToken{}))
	assert.Zero(t, count(db, &models.PasswordResetToken{}))
	assert.NoError(t, repo.CreateUser(&models.User{Email: "camille@example.com", Password: "hash"}), "the address is free again")
}

// Accounts an earlier version only marked as deleted are erased at start-up.
func TestPurgeDeletedUsers(t *testing.T) {
	repo, db := erasureDB(t)
	gone := userWithTraces(t, repo, db, "gone@example.com")
	userWithTraces(t, repo, db, "kept@example.com")
	require.NoError(t, db.Where("id = ?", gone).Delete(&models.User{}).Error) // old soft delete

	purged, err := repo.PurgeDeletedUsers()
	require.NoError(t, err)
	assert.Equal(t, int64(1), purged)
	assert.Equal(t, int64(1), count(db, &models.User{}))
	assert.Equal(t, int64(1), count(db, &models.RefreshToken{}), "the other account keeps its session")

	purged, err = repo.PurgeDeletedUsers()
	require.NoError(t, err)
	assert.Zero(t, purged, "idempotent")
}
