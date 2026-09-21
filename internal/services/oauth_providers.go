package services

import (
	"auth-service/internal/models"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"
	"time"
)

// ─── Registre des fournisseurs ──────────────────────

// providerConfig décrit un fournisseur OIDC. Google et Microsoft suivent le
// même standard : seuls les URLs et le libellé changent.
type providerConfig struct {
	Label        string
	AuthURL      string
	TokenURL     string
	UserInfoURL  string
	Scopes       string
	ClientID     string
	ClientSecret string
	// AssumeEmailVerified sert aux fournisseurs qui ne renvoient pas le claim
	// `email_verified` alors qu'ils garantissent l'email par ailleurs. Ce flag
	// conditionne le rattachement à un compte existant : ne l'activer que pour
	// un fournisseur de confiance.
	AssumeEmailVerified bool
}

var providerCatalog = map[string]func() providerConfig{
	"google": func() providerConfig {
		return providerConfig{
			Label:        "Google",
			AuthURL:      "https://accounts.google.com/o/oauth2/v2/auth",
			TokenURL:     "https://oauth2.googleapis.com/token",
			UserInfoURL:  "https://openidconnect.googleapis.com/v1/userinfo",
			Scopes:       "openid email profile",
			ClientID:     os.Getenv("GOOGLE_CLIENT_ID"),
			ClientSecret: os.Getenv("GOOGLE_CLIENT_SECRET"),
		}
	},
	"microsoft": func() providerConfig {
		return providerConfig{
			Label:       "Microsoft",
			AuthURL:     "https://login.microsoftonline.com/common/oauth2/v2.0/authorize",
			TokenURL:    "https://login.microsoftonline.com/common/oauth2/v2.0/token",
			UserInfoURL: "https://graph.microsoft.com/oidc/userinfo",
			Scopes:      "openid email profile",
			// L'endpoint OIDC de Microsoft ne renvoie pas `email_verified` :
			// sans ce flag, aucune connexion Microsoft ne pourrait jamais
			// rejoindre un compte existant.
			AssumeEmailVerified: true,
			ClientID:            os.Getenv("MICROSOFT_CLIENT_ID"),
			ClientSecret:        os.Getenv("MICROSOFT_CLIENT_SECRET"),
		}
	},
}

// ─── Service ────────────────────────────────────────

type OAuthService interface {
	// EnabledProviders ne renvoie que les fournisseurs dont les identifiants
	// sont configurés — le front n'affiche donc jamais un bouton inutilisable.
	EnabledProviders() []models.OAuthProviderInfo
	IsEnabled(provider string) bool
	AuthorizeURL(provider, state string) (string, error)
	// ExchangeCode échange le code d'autorisation contre le profil du client.
	ExchangeCode(provider, code string) (*models.OAuthProfile, error)
}

type oauthService struct {
	http *http.Client
}

func NewOAuthService() OAuthService {
	return &oauthService{http: &http.Client{Timeout: 10 * time.Second}}
}

func (s *oauthService) config(provider string) (providerConfig, error) {
	build, ok := providerCatalog[provider]
	if !ok {
		return providerConfig{}, fmt.Errorf("fournisseur inconnu : %s", provider)
	}
	cfg := build()
	if cfg.ClientID == "" || cfg.ClientSecret == "" {
		return providerConfig{}, fmt.Errorf("fournisseur %s non configuré", provider)
	}
	return cfg, nil
}

func (s *oauthService) IsEnabled(provider string) bool {
	_, err := s.config(provider)
	return err == nil
}

func (s *oauthService) EnabledProviders() []models.OAuthProviderInfo {
	enabled := []models.OAuthProviderInfo{}
	// Ordre stable : la map Go n'a pas d'ordre garanti.
	for _, name := range []string{"google", "microsoft"} {
		if cfg, err := s.config(name); err == nil {
			enabled = append(enabled, models.OAuthProviderInfo{Provider: name, Label: cfg.Label})
		}
	}
	return enabled
}

// RedirectURI doit correspondre exactement à celle déclarée chez le fournisseur.
func RedirectURI(provider string) string {
	base := os.Getenv("OAUTH_REDIRECT_BASE_URL")
	if base == "" {
		base = "http://localhost:3000"
	}
	return fmt.Sprintf("%s/api/auth/oauth/%s/callback", strings.TrimRight(base, "/"), provider)
}

func (s *oauthService) AuthorizeURL(provider, state string) (string, error) {
	cfg, err := s.config(provider)
	if err != nil {
		return "", err
	}

	params := url.Values{}
	params.Set("client_id", cfg.ClientID)
	params.Set("redirect_uri", RedirectURI(provider))
	params.Set("response_type", "code")
	params.Set("scope", cfg.Scopes)
	params.Set("state", state)
	// Force le choix du compte plutôt qu'une reconnexion silencieuse.
	params.Set("prompt", "select_account")

	return cfg.AuthURL + "?" + params.Encode(), nil
}

type tokenResponse struct {
	AccessToken      string `json:"access_token"`
	Error            string `json:"error"`
	ErrorDescription string `json:"error_description"`
}

type userInfoResponse struct {
	Sub               string `json:"sub"`
	Email             string `json:"email"`
	EmailVerified     any    `json:"email_verified"`
	Name              string `json:"name"`
	PreferredUsername string `json:"preferred_username"`
}

func (s *oauthService) ExchangeCode(provider, code string) (*models.OAuthProfile, error) {
	cfg, err := s.config(provider)
	if err != nil {
		return nil, err
	}

	form := url.Values{}
	form.Set("client_id", cfg.ClientID)
	form.Set("client_secret", cfg.ClientSecret)
	form.Set("code", code)
	form.Set("grant_type", "authorization_code")
	form.Set("redirect_uri", RedirectURI(provider))

	resp, err := s.http.PostForm(cfg.TokenURL, form)
	if err != nil {
		return nil, fmt.Errorf("échange du code impossible : %w", err)
	}
	defer resp.Body.Close()

	var token tokenResponse
	if err := json.NewDecoder(resp.Body).Decode(&token); err != nil {
		return nil, errors.New("réponse illisible du fournisseur")
	}
	if token.Error != "" {
		return nil, fmt.Errorf("%s : %s", token.Error, token.ErrorDescription)
	}
	if token.AccessToken == "" {
		return nil, errors.New("aucun jeton renvoyé par le fournisseur")
	}

	profile, err := s.fetchUserInfo(cfg, token.AccessToken)
	if err != nil {
		return nil, err
	}
	profile.Provider = provider
	return profile, nil
}

func (s *oauthService) fetchUserInfo(cfg providerConfig, accessToken string) (*models.OAuthProfile, error) {
	req, err := http.NewRequest(http.MethodGet, cfg.UserInfoURL, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("Authorization", "Bearer "+accessToken)

	resp, err := s.http.Do(req)
	if err != nil {
		return nil, fmt.Errorf("profil illisible : %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("le fournisseur a renvoyé %d", resp.StatusCode)
	}

	var info userInfoResponse
	if err := json.NewDecoder(resp.Body).Decode(&info); err != nil {
		return nil, errors.New("profil illisible")
	}

	email := info.Email
	if email == "" {
		email = info.PreferredUsername // Microsoft renvoie parfois l'email ici
	}
	if info.Sub == "" || email == "" {
		return nil, errors.New("le fournisseur n'a pas renvoyé d'email")
	}

	return &models.OAuthProfile{
		Subject:       info.Sub,
		Email:         strings.ToLower(email),
		EmailVerified: cfg.AssumeEmailVerified || isVerified(info.EmailVerified),
		Name:          info.Name,
	}, nil
}

// isVerified normalise email_verified : Google l'envoie en booléen, d'autres
// fournisseurs en chaîne "true".
func isVerified(raw any) bool {
	switch value := raw.(type) {
	case bool:
		return value
	case string:
		return value == "true"
	default:
		return false
	}
}
