package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"auth-service/internal/tokens"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func TestRefreshAccessToken_RotatesStoredHash(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))
	old, _ := tokens.MustFromEnv().IssueRefresh("U1", time.Now())
	hash := tokens.Hash(old)

	repo.On("GetRefreshToken", hash).Return(&models.RefreshToken{UserID: "U1"}, nil).Once()
	repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1", Email: "u@t.com"}, nil).Once()
	repo.On("RevokeRefreshToken", hash).Return(nil).Once()
	repo.On("GetUserRoles", "U1").Return([]models.Role{}, nil).Once()
	repo.On("CreateRefreshToken", mock.MatchedBy(func(rt *models.RefreshToken) bool {
		return rt.UserID == "U1" && len(rt.Token) == 64 // stored as a sha256 hex digest
	})).Return(nil).Once()

	access, refresh, err := srv.RefreshAccessToken(old)

	assert.NoError(t, err)
	assert.NotEqual(t, old, refresh)
	_, parseErr := tokens.MustFromEnv().ParseAccess(access)
	assert.NoError(t, parseErr)
	repo.AssertExpectations(t)
}

func TestRefreshAccessToken_RejectsAccessToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))
	access, _ := tokens.MustFromEnv().IssueAccess("U1", nil, time.Now())

	_, _, err := srv.RefreshAccessToken(access)

	assert.Error(t, err)
	repo.AssertNotCalled(t, "GetRefreshToken", mock.Anything)
}

func TestRefreshAccessToken_RejectsRevokedToken(t *testing.T) {
	t.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))
	old, _ := tokens.MustFromEnv().IssueRefresh("U1", time.Now())
	repo.On("GetRefreshToken", tokens.Hash(old)).Return(nil, errors.New("record not found")).Once()

	_, _, err := srv.RefreshAccessToken(old)

	assert.Error(t, err)
	repo.AssertNotCalled(t, "CreateRefreshToken", mock.Anything)
}

func TestLogoutUser_RevokesByHash(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))
	repo.On("RevokeRefreshToken", tokens.Hash("rt")).Return(nil).Once()

	assert.NoError(t, srv.LogoutUser("rt"))
	repo.AssertExpectations(t)
}
