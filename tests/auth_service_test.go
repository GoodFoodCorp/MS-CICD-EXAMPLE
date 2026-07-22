package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"errors"
	"os"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"golang.org/x/crypto/bcrypt"
)

func strPtr(s string) *string { return &s }

// ═══════════════════════════════════════════════════════
// AuthService Tests
// ═══════════════════════════════════════════════════════

func TestAuthService_PasswordResetFlow(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	email := new(MockEmailService)
	srv := services.NewAuthService(repo, email)

	t.Run("ForgotPassword", func(t *testing.T) {
		repo.On("FindGlobalByEmail", "f@t.com").Return(&models.User{ID: "U1", Email: "f@t.com"}, nil).Once()
		repo.On("CreatePasswordResetToken", mock.Anything).Return(nil).Once()
		email.On("SendPasswordResetEmail", mock.Anything, mock.Anything).Return(nil).Once()

		_, err := srv.ForgotPassword(&models.ForgotPasswordRequest{Email: "f@t.com"})
		assert.NoError(t, err)
		time.Sleep(50 * time.Millisecond)
	})

	t.Run("ResetPassword", func(t *testing.T) {
		oldHash, _ := bcrypt.GenerateFromPassword([]byte("OldPass123"), bcrypt.DefaultCost)
		req := &models.ResetPasswordRequest{Token: "tok", NewPassword: "NewPass123", ConfirmPassword: "NewPass123"}
		repo.On("GetPasswordResetToken", "tok").Return(&models.PasswordResetToken{UserID: "U1"}, nil).Once()
		repo.On("FindUserByID", "U1").Return(&models.User{Password: string(oldHash)}, nil).Once()
		repo.On("UpdateUserPassword", "U1", mock.Anything).Return(nil).Once()
		repo.On("MarkPasswordResetTokenAsUsed", "tok").Return(nil).Once()
		assert.NoError(t, srv.ResetPassword(req))
	})
}

func TestAuthService_VerifyEmail(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	email := new(MockEmailService)
	srv := services.NewAuthService(repo, email)

	vt := &models.EmailVerificationToken{UserID: "U1", ID: 1}
	repo.On("FindEmailVerificationToken", "tok").Return(vt, nil).Once()
	repo.On("MarkUserAsVerified", "U1").Return(nil).Once()
	repo.On("DeleteEmailVerificationToken", uint(1)).Return(nil).Once()

	err := srv.VerifyEmail("tok")
	assert.NoError(t, err)
}

// ═══════════════════════════════════════════════════════
// UserAdminService Tests
// ═══════════════════════════════════════════════════════

func TestUserAdminService_GetAllUsers(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewUserAdminService(repo)

	repo.On("FindAll", 1, 10).Return([]models.User{{ID: "U1"}}, int64(1), nil).Once()
	users, total, err := srv.GetAllUsers(1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, users, 1)
}

func TestUserAdminService_GetUserByID(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewUserAdminService(repo)

	repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1"}, nil).Once()
	u, err := srv.GetUserByID("U1")
	assert.NoError(t, err)
	assert.Equal(t, "U1", u.ID)
}

func TestUserAdminService_GetUserByEmail(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewUserAdminService(repo)

	repo.On("FindGlobalByEmail", "e@e.com").Return(&models.User{Email: "e@e.com"}, nil).Once()
	u, err := srv.GetUserByEmail("e@e.com")
	assert.NoError(t, err)
	assert.Equal(t, "e@e.com", u.Email)
}

func TestRoleService_CreateRole(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewRoleService(repo)

	t.Run("Success", func(t *testing.T) {
		repo.On("FindRoleByName", "editor").Return(nil, errors.New("not found")).Once()
		repo.On("CreateRole", mock.Anything).Return(nil).Once()
		assert.NoError(t, srv.CreateRole("editor", "Can edit"))
	})

	t.Run("AlreadyExists", func(t *testing.T) {
		repo.On("FindRoleByName", "admin").Return(&models.Role{}, nil).Once()
		err := srv.CreateRole("admin", "Boss")
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "existe déjà")
	})
}

func TestRoleService_RemoveRoleFromUser(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewRoleService(repo)

	repo.On("RemoveRoleFromUser", "U1", uint(2)).Return(nil).Once()
	assert.NoError(t, srv.RemoveRoleFromUser("U1", 2))
}

func TestRoleService_GetUserRoles(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewRoleService(repo)

	repo.On("GetUserRoles", "U1").Return([]models.Role{{ID: 1, Name: "admin"}}, nil).Once()
	roles, err := srv.GetUserRoles("U1")
	assert.NoError(t, err)
	assert.Len(t, roles, 1)
}
