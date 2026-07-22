package tests

import (
	"auth-service/internal/models"

	"github.com/stretchr/testify/mock"
)

// ─── Repository Mock ────────────────────────────────

type MockAuthRepository struct {
	mock.Mock
}

func (m *MockAuthRepository) CreateUser(user *models.User) error {
	return m.Called(user).Error(0)
}

func (m *MockAuthRepository) FindByEmail(email, tenantID string) (*models.User, error) {
	args := m.Called(email, tenantID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) FindGlobalByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) FindUserByID(id string) (*models.User, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockAuthRepository) UpdateUserPassword(userID string, hashed string) error {
	return m.Called(userID, hashed).Error(0)
}

func (m *MockAuthRepository) FindAll(page, limit int) ([]models.User, int64, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockAuthRepository) MarkUserAsVerified(userID string) error {
	return m.Called(userID).Error(0)
}

func (m *MockAuthRepository) CreateEmailVerificationToken(t *models.EmailVerificationToken) error {
	return m.Called(t).Error(0)
}

func (m *MockAuthRepository) FindEmailVerificationToken(token string) (*models.EmailVerificationToken, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.EmailVerificationToken), args.Error(1)
}

func (m *MockAuthRepository) DeleteEmailVerificationToken(id uint) error {
	return m.Called(id).Error(0)
}

func (m *MockAuthRepository) CreatePasswordResetToken(t *models.PasswordResetToken) error {
	return m.Called(t).Error(0)
}

func (m *MockAuthRepository) GetPasswordResetToken(token string) (*models.PasswordResetToken, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.PasswordResetToken), args.Error(1)
}

func (m *MockAuthRepository) MarkPasswordResetTokenAsUsed(token string) error {
	return m.Called(token).Error(0)
}

func (m *MockAuthRepository) CreateRefreshToken(rt *models.RefreshToken) error {
	return m.Called(rt).Error(0)
}

func (m *MockAuthRepository) GetRefreshToken(token string) (*models.RefreshToken, error) {
	args := m.Called(token)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.RefreshToken), args.Error(1)
}

func (m *MockAuthRepository) RevokeRefreshToken(token string) error {
	return m.Called(token).Error(0)
}

func (m *MockAuthRepository) CreateRole(role *models.Role) error {
	return m.Called(role).Error(0)
}

func (m *MockAuthRepository) FindRoleByName(name string) (*models.Role, error) {
	args := m.Called(name)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockAuthRepository) FindRoleBySlug(slug string) (*models.Role, error) {
	args := m.Called(slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockAuthRepository) FindRoleByID(id uint) (*models.Role, error) {
	args := m.Called(id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}

func (m *MockAuthRepository) AssignRoleToUser(uID string, rID uint) error {
	return m.Called(uID, rID).Error(0)
}

func (m *MockAuthRepository) RemoveRoleFromUser(uID string, rID uint) error {
	return m.Called(uID, rID).Error(0)
}

func (m *MockAuthRepository) GetUserRoles(userID string) ([]models.Role, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockAuthRepository) GetAllRoles() ([]models.Role, error) {
	args := m.Called()
	return args.Get(0).([]models.Role), args.Error(1)
}

// ─── Email Service Mock ─────────────────────────────

type MockEmailService struct {
	mock.Mock
}

func (m *MockEmailService) SendVerificationEmail(e, t string) error {
	return m.Called(e, t).Error(0)
}

func (m *MockEmailService) SendPasswordResetEmail(e, t string) error {
	return m.Called(e, t).Error(0)
}

// ─── AuthService Mock (pour AuthController) ─────────

type MockAuthService struct {
	mock.Mock
}

func (m *MockAuthService) RegisterUser(req *models.RegisterRequest) (string, error) {
	args := m.Called(req)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) LoginUser(req *models.LoginRequest) (string, string, error) {
	args := m.Called(req)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) LogoutUser(refreshToken string) error {
	return m.Called(refreshToken).Error(0)
}

func (m *MockAuthService) RefreshAccessToken(oldRefreshToken string) (string, string, error) {
	args := m.Called(oldRefreshToken)
	return args.String(0), args.String(1), args.Error(2)
}

func (m *MockAuthService) VerifyEmail(token string) error {
	return m.Called(token).Error(0)
}

func (m *MockAuthService) ForgotPassword(req *models.ForgotPasswordRequest) (string, error) {
	args := m.Called(req)
	return args.String(0), args.Error(1)
}

func (m *MockAuthService) ResetPassword(req *models.ResetPasswordRequest) error {
	return m.Called(req).Error(0)
}

// ─── UserAdminService Mock (pour AdminController) ───

type MockUserAdminService struct {
	mock.Mock
}

func (m *MockUserAdminService) GetAllUsers(page int, limit int) ([]models.User, int64, error) {
	args := m.Called(page, limit)
	return args.Get(0).([]models.User), args.Get(1).(int64), args.Error(2)
}

func (m *MockUserAdminService) GetUserByID(userID string) (*models.User, error) {
	args := m.Called(userID)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserAdminService) GetUserByEmail(email string) (*models.User, error) {
	args := m.Called(email)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.User), args.Error(1)
}

func (m *MockUserAdminService) PromoteUserToAdmin(userID string) error {
	return m.Called(userID).Error(0)
}

// ─── RoleService Mock (pour RoleController) ─────────

type MockRoleService struct {
	mock.Mock
}

func (m *MockRoleService) CreateRole(name, description string) error {
	return m.Called(name, description).Error(0)
}

func (m *MockRoleService) AssignRoleToUser(userID string, roleID uint) error {
	return m.Called(userID, roleID).Error(0)
}

func (m *MockRoleService) RemoveRoleFromUser(userID string, roleID uint) error {
	return m.Called(userID, roleID).Error(0)
}

func (m *MockRoleService) GetUserRoles(userID string) ([]models.Role, error) {
	args := m.Called(userID)
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockRoleService) GetAllRoles() ([]models.Role, error) {
	args := m.Called()
	return args.Get(0).([]models.Role), args.Error(1)
}

func (m *MockRoleService) GetRoleBySlug(slug string) (*models.Role, error) {
	args := m.Called(slug)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*models.Role), args.Error(1)
}
