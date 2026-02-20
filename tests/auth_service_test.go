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

func TestAuthService_Register(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	email := new(MockEmailService)
	srv := services.NewAuthService(repo, email)

	t.Run("PasswordTooShort", func(t *testing.T) {
		_, err := srv.RegisterUser(&models.RegisterRequest{Password: "short"})
		assert.Error(t, err)
	})

	t.Run("EmailAlreadyUsed", func(t *testing.T) {
		req := &models.RegisterRequest{Email: "reg@test.com", Password: "ValidPass123", TenantID: "T1"}
		repo.On("FindGlobalByEmail", req.Email).Return(&models.User{}, nil).Once()
		_, err := srv.RegisterUser(req)
		assert.Equal(t, "cet email est déjà utilisé", err.Error())
	})

	t.Run("Success", func(t *testing.T) {
		req := &models.RegisterRequest{Email: "reg@test.com", Password: "ValidPass123", TenantID: "T1"}
		repo.On("FindGlobalByEmail", req.Email).Return(nil, errors.New("not found")).Once()
		repo.On("CreateUser", mock.Anything).Return(nil).Once()
		repo.On("CreateEmailVerificationToken", mock.Anything).Return(nil).Once()
		email.On("SendVerificationEmail", req.Email, mock.Anything).Return(nil).Once()

		_, err := srv.RegisterUser(req)
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	})

	t.Run("SuccessWithoutTenant", func(t *testing.T) {
		req := &models.RegisterRequest{Email: "customer@test.com", Password: "ValidPass123"}
		repo.On("FindGlobalByEmail", req.Email).Return(nil, errors.New("not found")).Once()
		repo.On("CreateUser", mock.Anything).Return(nil).Once()
		repo.On("FindRoleBySlug", "user").Return(&models.Role{ID: 2, Name: "user", Slug: "user"}, nil).Once()
		repo.On("AssignRoleToUser", mock.Anything, uint(2)).Return(nil).Once()
		repo.On("CreateEmailVerificationToken", mock.Anything).Return(nil).Once()
		email.On("SendVerificationEmail", req.Email, mock.Anything).Return(nil).Once()

		_, err := srv.RegisterUser(req)
		assert.NoError(t, err)
		time.Sleep(10 * time.Millisecond)
	})
}

func TestAuthService_LoginAndRefresh(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	email := new(MockEmailService)
	srv := services.NewAuthService(repo, email)

	pass := "Pass1234"
	hash, _ := bcrypt.GenerateFromPassword([]byte(pass), bcrypt.DefaultCost)
	user := &models.User{ID: "U1", Email: "l@t.com", Password: string(hash), IsEmailVerified: true, TenantID: strPtr("T1")}

	t.Run("LoginSuccess", func(t *testing.T) {
		req := &models.LoginRequest{Email: "l@t.com", Password: pass}
		repo.On("FindGlobalByEmail", req.Email).Return(user, nil).Once()
		repo.On("GetUserRoles", "U1").Return([]models.Role{}, nil).Once()
		repo.On("CreateRefreshToken", mock.Anything).Return(nil).Once()

		at, rt, err := srv.LoginUser(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, at)
		assert.NotEmpty(t, rt)
	})

	t.Run("RefreshSuccess", func(t *testing.T) {
		// On a besoin d'un vrai RT pour refresh, re-login
		req := &models.LoginRequest{Email: "l@t.com", Password: pass}
		repo.On("FindGlobalByEmail", req.Email).Return(user, nil).Once()
		repo.On("GetUserRoles", "U1").Return([]models.Role{}, nil).Once()
		repo.On("CreateRefreshToken", mock.Anything).Return(nil).Once()
		_, rt, _ := srv.LoginUser(req)

		repo.On("GetRefreshToken", rt).Return(&models.RefreshToken{UserID: "U1"}, nil).Once()
		repo.On("FindUserByID", "U1").Return(user, nil).Once()
		repo.On("RevokeRefreshToken", rt).Return(nil).Once()
		repo.On("GetUserRoles", "U1").Return([]models.Role{}, nil).Once()
		repo.On("CreateRefreshToken", mock.Anything).Return(nil).Once()

		newAt, newRt, err := srv.RefreshAccessToken(rt)
		assert.NoError(t, err)
		assert.NotEmpty(t, newAt)
		assert.NotEmpty(t, newRt)
	})

	t.Run("LogoutSuccess", func(t *testing.T) {
		repo.On("RevokeRefreshToken", "token").Return(nil).Once()
		assert.NoError(t, srv.LogoutUser("token"))
	})
}

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

func TestUserAdminService_PromoteUserToAdmin(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewUserAdminService(repo)

	t.Run("RoleExists", func(t *testing.T) {
		repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1", TenantID: strPtr("T1")}, nil).Once()
		repo.On("FindRoleByName", "admin").Return(&models.Role{ID: 1, Name: "admin"}, nil).Once()
		repo.On("AssignRoleToUser", "U1", uint(1)).Return(nil).Once()

		err := srv.PromoteUserToAdmin("U1")
		assert.NoError(t, err)
	})

	t.Run("RoleCreatedIfMissing", func(t *testing.T) {
		repo.On("FindUserByID", "U2").Return(&models.User{ID: "U2", TenantID: strPtr("T2")}, nil).Once()
		repo.On("FindRoleByName", "admin").Return(nil, errors.New("not found")).Once()
		repo.On("CreateRole", mock.Anything).Return(nil).Once()
		repo.On("FindRoleByName", "admin").Return(&models.Role{ID: 2, Name: "admin"}, nil).Once()
		repo.On("AssignRoleToUser", "U2", uint(2)).Return(nil).Once()

		err := srv.PromoteUserToAdmin("U2")
		assert.NoError(t, err)
	})
}

// ═══════════════════════════════════════════════════════
// RoleService Tests
// ═══════════════════════════════════════════════════════

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

func TestRoleService_AssignRoleToUser(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewRoleService(repo)

	t.Run("Success", func(t *testing.T) {
		repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1", TenantID: strPtr("T1")}, nil).Once()
		repo.On("FindRoleByID", uint(2)).Return(&models.Role{ID: 2}, nil).Once()
		repo.On("AssignRoleToUser", "U1", uint(2)).Return(nil).Once()
		assert.NoError(t, srv.AssignRoleToUser("U1", 2))
	})

	t.Run("UserNotFound", func(t *testing.T) {
		repo.On("FindUserByID", "UNKNOWN").Return(nil, errors.New("not found")).Once()
		err := srv.AssignRoleToUser("UNKNOWN", 2)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "utilisateur introuvable")
	})

	t.Run("RoleNotFound", func(t *testing.T) {
		repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1", TenantID: strPtr("T1")}, nil).Once()
		repo.On("FindRoleByID", uint(99)).Return(nil, errors.New("not found")).Once()
		err := srv.AssignRoleToUser("U1", 99)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "rôle introuvable")
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

func TestRoleService_GetAllRoles(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewRoleService(repo)

	repo.On("GetAllRoles").Return([]models.Role{{ID: 1, Name: "admin"}}, nil).Once()
	roles, err := srv.GetAllRoles()
	assert.NoError(t, err)
	assert.Len(t, roles, 1)
}

// ═══════════════════════════════════════════════════════
// TenantService Tests
// ═══════════════════════════════════════════════════════

func TestTenantService_CreateTenant(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("CreateTenant", mock.Anything).Return(nil).Once()
	tenant, err := srv.CreateTenant(&models.CreateTenantRequest{Name: "My Corp", Plan: "basic"})
	assert.NoError(t, err)
	assert.Equal(t, "My Corp", tenant.Name)
	assert.NotEmpty(t, tenant.Slug)
}

func TestTenantService_GetTenantByID(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("FindTenantByID", "T1").Return(&models.Tenant{ID: "T1"}, nil).Once()
	tenant, err := srv.GetTenantByID("T1")
	assert.NoError(t, err)
	assert.Equal(t, "T1", tenant.ID)
}

func TestTenantService_GetTenantBySlug(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("FindTenantBySlug", "test").Return(&models.Tenant{Slug: "test"}, nil).Once()
	tenant, err := srv.GetTenantBySlug("test")
	assert.NoError(t, err)
	assert.Equal(t, "test", tenant.Slug)
}

func TestTenantService_GetAllTenants(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("GetAllTenants", 1, 10).Return([]models.Tenant{{ID: "T1"}}, int64(1), nil).Once()
	tenants, total, err := srv.GetAllTenants(1, 10)
	assert.NoError(t, err)
	assert.Equal(t, int64(1), total)
	assert.Len(t, tenants, 1)
}

func TestTenantService_UpdateTenant(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("UpdateTenant", "T1", mock.Anything).Return(nil).Once()
	err := srv.UpdateTenant("T1", &models.UpdateTenantRequest{Name: "Updated"})
	assert.NoError(t, err)
}

func TestTenantService_DeleteTenant(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewTenantService(repo)

	repo.On("DeleteTenant", "T1").Return(nil).Once()
	assert.NoError(t, srv.DeleteTenant("T1"))
}