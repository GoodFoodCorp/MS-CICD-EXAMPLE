package controllers

import (
	"auth-service/internal/services"
	"crypto/rand"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
)

const (
	stateCookie    = "oauth_state"
	stateCookieTTL = 600 // 10 minutes, le temps de l'aller-retour
)

type OAuthController struct {
	oauth services.OAuthService
	auth  services.AuthService
}

func NewOAuthController(oauth services.OAuthService, auth services.AuthService) *OAuthController {
	return &OAuthController{oauth: oauth, auth: auth}
}

// @Summary      Liste des fournisseurs OAuth activés
// @Description  Ne renvoie que les fournisseurs dont les identifiants sont configurés
// @Tags         OAuth
// @Produce      json
// @Success      200 {array} models.OAuthProviderInfo
// @Router       /api/auth/oauth/providers [get]
func (ctrl *OAuthController) Providers(c *gin.Context) {
	c.JSON(http.StatusOK, ctrl.oauth.EnabledProviders())
}

// @Summary      Démarre la connexion via un fournisseur
// @Description  Redirige vers Google/Microsoft avec un state anti-CSRF
// @Tags         OAuth
// @Param        provider path string true "google | microsoft"
// @Success      302 {string} string "Redirection"
// @Router       /api/auth/oauth/{provider} [get]
func (ctrl *OAuthController) Start(c *gin.Context) {
	provider := c.Param("provider")
	if !ctrl.oauth.IsEnabled(provider) {
		c.JSON(http.StatusNotFound, gin.H{"error": "Fournisseur non disponible"})
		return
	}

	state, err := randomState()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Erreur interne"})
		return
	}
	authorizeURL, err := ctrl.oauth.AuthorizeURL(provider, state)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Le state est rejoué au retour : il doit survivre à la redirection mais
	// rester inaccessible au JavaScript.
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(stateCookie, state, stateCookieTTL, "/", "", isSecureCookie(), true)

	c.Redirect(http.StatusFound, authorizeURL)
}

// @Summary      Retour du fournisseur OAuth
// @Description  Échange le code, connecte ou crée le compte, puis renvoie vers le front
// @Tags         OAuth
// @Param        provider path string true "google | microsoft"
// @Param        code query string true "Code d'autorisation"
// @Param        state query string true "State anti-CSRF"
// @Success      302 {string} string "Redirection vers le front"
// @Router       /api/auth/oauth/{provider}/callback [get]
func (ctrl *OAuthController) Callback(c *gin.Context) {
	provider := c.Param("provider")

	if providerError := c.Query("error"); providerError != "" {
		ctrl.redirectWithError(c, providerError)
		return
	}

	expected, err := c.Cookie(stateCookie)
	if err != nil || expected == "" || expected != c.Query("state") {
		ctrl.redirectWithError(c, "state_invalide")
		return
	}
	// Un state ne sert qu'une fois.
	c.SetCookie(stateCookie, "", -1, "/", "", isSecureCookie(), true)

	code := c.Query("code")
	if code == "" {
		ctrl.redirectWithError(c, "code_manquant")
		return
	}

	profile, err := ctrl.oauth.ExchangeCode(provider, code)
	if err != nil {
		ctrl.redirectWithError(c, "echange_impossible")
		return
	}

	accessToken, refreshToken, err := ctrl.auth.LoginWithOAuth(profile)
	if err != nil {
		ctrl.redirectWithError(c, err.Error())
		return
	}

	// Les jetons partent dans le fragment (#) et non la query : un fragment
	// n'est jamais transmis au serveur ni écrit dans ses logs d'accès.
	c.Redirect(http.StatusFound, fmt.Sprintf(
		"%s/oauth/callback#access_token=%s&refresh_token=%s",
		frontendURL(),
		url.QueryEscape(accessToken),
		url.QueryEscape(refreshToken),
	))
}

func (ctrl *OAuthController) redirectWithError(c *gin.Context, reason string) {
	c.Redirect(http.StatusFound, fmt.Sprintf(
		"%s/oauth/callback#error=%s", frontendURL(), url.QueryEscape(reason),
	))
}

func frontendURL() string {
	if value := os.Getenv("FRONTEND_URL"); value != "" {
		return strings.TrimRight(value, "/")
	}
	return "http://localhost:3000"
}

// isSecureCookie : le cookie n'est marqué Secure qu'en HTTPS, sinon le
// navigateur le refuserait en développement sur http://localhost.
func isSecureCookie() bool {
	return strings.HasPrefix(frontendURL(), "https://")
}

func randomState() (string, error) {
	buffer := make([]byte, 32)
	if _, err := rand.Read(buffer); err != nil {
		return "", err
	}
	return hex.EncodeToString(buffer), nil
}
