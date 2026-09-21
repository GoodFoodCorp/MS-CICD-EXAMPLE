package tests

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"errors"
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

func googleProfile(email string, verified bool) *models.OAuthProfile {
	return &models.OAuthProfile{
		Provider:      "google",
		Subject:       "google-sub-1",
		Email:         email,
		EmailVerified: verified,
		Name:          "Marie Dupont",
	}
}

// expectTokenIssuance couvre les appels faits par generateTokens.
func expectTokenIssuance(repo *MockAuthRepository, userID string) {
	repo.On("GetUserRoles", userID).Return([]models.Role{{ID: 1, Name: "Client", Slug: "user"}}, nil).Once()
	repo.On("CreateRefreshToken", mock.Anything).Return(nil).Once()
}

func TestOAuthLogin_KnownAccountSignsInDirectly(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))

	repo.On("FindOAuthAccount", "google", "google-sub-1").
		Return(&models.OAuthAccount{UserID: "U1", Provider: "google"}, nil).Once()
	repo.On("FindUserByID", "U1").Return(&models.User{ID: "U1", Email: "m@t.com"}, nil).Once()
	expectTokenIssuance(repo, "U1")

	access, refresh, err := srv.LoginWithOAuth(googleProfile("m@t.com", true))

	assert.NoError(t, err)
	assert.NotEmpty(t, access)
	assert.NotEmpty(t, refresh)
	repo.AssertExpectations(t)
	// Aucun compte n'est créé pour un utilisateur déjà connu.
	repo.AssertNotCalled(t, "CreateUser", mock.Anything)
}

func TestOAuthLogin_LinksToExistingAccountWhenEmailIsVerified(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))

	repo.On("FindOAuthAccount", "google", "google-sub-1").Return(nil, errors.New("not found")).Once()
	repo.On("FindGlobalByEmail", "m@t.com").Return(&models.User{ID: "U1", Email: "m@t.com"}, nil).Once()
	repo.On("CreateOAuthAccount", mock.Anything).Return(nil).Once()
	expectTokenIssuance(repo, "U1")

	_, _, err := srv.LoginWithOAuth(googleProfile("m@t.com", true))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
	repo.AssertNotCalled(t, "CreateUser", mock.Anything)
}

// Sécurité : sans email vérifié, un fournisseur laxiste permettrait de
// s'emparer d'un compte existant en déclarant son adresse.
func TestOAuthLogin_RefusesToHijackAnAccountWithAnUnverifiedEmail(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))

	repo.On("FindOAuthAccount", "google", "google-sub-1").Return(nil, errors.New("not found")).Once()
	repo.On("FindGlobalByEmail", "victime@t.com").Return(&models.User{ID: "U1", Email: "victime@t.com"}, nil).Once()

	_, _, err := srv.LoginWithOAuth(googleProfile("victime@t.com", false))

	assert.ErrorContains(t, err, "déjà utilisé")
	repo.AssertNotCalled(t, "CreateOAuthAccount", mock.Anything)
	repo.AssertNotCalled(t, "CreateRefreshToken", mock.Anything)
}

func TestOAuthLogin_CreatesAVerifiedClientAccountOnFirstLogin(t *testing.T) {
	os.Setenv("JWT_SECRET", "test_secret")
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))

	repo.On("FindOAuthAccount", "google", "google-sub-1").Return(nil, errors.New("not found")).Once()
	repo.On("FindGlobalByEmail", "neuf@t.com").Return(nil, errors.New("not found")).Once()
	repo.On("CreateUser", mock.MatchedBy(func(user *models.User) bool {
		user.ID = "U9" // GORM le ferait via BeforeCreate
		// Le compte est vérifié d'emblée et reçoit un mot de passe inutilisable.
		return user.Email == "neuf@t.com" && user.IsEmailVerified && user.Password != ""
	})).Return(nil).Once()
	repo.On("FindRoleBySlug", "user").Return(&models.Role{ID: 1, Slug: "user"}, nil).Once()
	repo.On("AssignRoleToUser", "U9", uint(1)).Return(nil).Once()
	repo.On("CreateOAuthAccount", mock.Anything).Return(nil).Once()
	expectTokenIssuance(repo, "U9")

	_, _, err := srv.LoginWithOAuth(googleProfile("neuf@t.com", true))

	assert.NoError(t, err)
	repo.AssertExpectations(t)
}

func TestOAuthLogin_RejectsAnIncompleteProfile(t *testing.T) {
	repo := new(MockAuthRepository)
	srv := services.NewAuthService(repo, new(MockEmailService))

	_, _, err := srv.LoginWithOAuth(&models.OAuthProfile{Provider: "google", Subject: "", Email: ""})
	assert.Error(t, err)
}

// ─── Registre des fournisseurs ──────────────────────

func TestOAuthProviders_OnlyConfiguredOnesAreExposed(t *testing.T) {
	os.Setenv("GOOGLE_CLIENT_ID", "gid")
	os.Setenv("GOOGLE_CLIENT_SECRET", "gsecret")
	os.Unsetenv("MICROSOFT_CLIENT_ID")
	os.Unsetenv("MICROSOFT_CLIENT_SECRET")
	defer os.Unsetenv("GOOGLE_CLIENT_ID")
	defer os.Unsetenv("GOOGLE_CLIENT_SECRET")

	srv := services.NewOAuthService()
	enabled := srv.EnabledProviders()

	assert.Len(t, enabled, 1)
	assert.Equal(t, "google", enabled[0].Provider)
	assert.True(t, srv.IsEnabled("google"))
	assert.False(t, srv.IsEnabled("microsoft"))
	assert.False(t, srv.IsEnabled("inconnu"))
}

func TestOAuthProviders_AuthorizeURLCarriesStateAndRedirect(t *testing.T) {
	os.Setenv("GOOGLE_CLIENT_ID", "gid")
	os.Setenv("GOOGLE_CLIENT_SECRET", "gsecret")
	os.Setenv("OAUTH_REDIRECT_BASE_URL", "http://localhost:3000")
	defer os.Unsetenv("GOOGLE_CLIENT_ID")
	defer os.Unsetenv("GOOGLE_CLIENT_SECRET")
	defer os.Unsetenv("OAUTH_REDIRECT_BASE_URL")

	srv := services.NewOAuthService()
	url, err := srv.AuthorizeURL("google", "state-123")

	assert.NoError(t, err)
	assert.Contains(t, url, "https://accounts.google.com/o/oauth2/v2/auth")
	assert.Contains(t, url, "client_id=gid")
	assert.Contains(t, url, "state=state-123")
	assert.Contains(t, url, "response_type=code")
	assert.Contains(t, url, "redirect_uri=http%3A%2F%2Flocalhost%3A3000%2Fapi%2Fauth%2Foauth%2Fgoogle%2Fcallback")
}

func TestOAuthProviders_UnconfiguredProviderCannotStartAFlow(t *testing.T) {
	os.Unsetenv("MICROSOFT_CLIENT_ID")
	os.Unsetenv("MICROSOFT_CLIENT_SECRET")

	_, err := services.NewOAuthService().AuthorizeURL("microsoft", "state")
	assert.ErrorContains(t, err, "non configuré")
}
