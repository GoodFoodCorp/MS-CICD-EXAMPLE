package tests

import (
	"auth-service/internal/middleware"
	"auth-service/internal/tokens"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

/* Helper pour générer un access token valide pour les tests */
func generateTestToken(secret string, userID string, roles []string) string {
	cfg := tokens.Config{AccessSecret: []byte(secret), RefreshSecret: []byte("refresh-" + secret),
		Issuer: tokens.DefaultIssuer, Audience: tokens.DefaultAudience}
	str, _ := cfg.IssueAccess(userID, map[string]any{"tenant_id": "T1", "roles": roles}, time.Now())
	return str
}

func TestAuthMiddleware(t *testing.T) {
	// Configuration de l'environnement de test
	os.Setenv("JWT_SECRET", "test_secret_key")
	gin.SetMode(gin.TestMode)

	t.Run("Missing_Cookie", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)

		// Appel du middleware directement
		middleware.AuthMiddleware()(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "token manquant")
	})

	t.Run("Invalid_Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: "invalid.token.garbage"})

		middleware.AuthMiddleware()(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Token invalide")
	})

	t.Run("Refresh_Token_Rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		refresh, _ := tokens.MustFromEnv().IssueRefresh("User123", time.Now())
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: refresh})

		middleware.AuthMiddleware()(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Legacy_Token_Without_Type_Rejected", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		legacy, _ := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
			"sub": "User123", "exp": time.Now().Add(time.Hour).Unix(),
		}).SignedString([]byte("test_secret_key"))
		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.Header.Set("Authorization", "Bearer "+legacy)

		middleware.AuthMiddleware()(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
	})

	t.Run("Valid_Token_Success", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Génération d'un vrai token
		token := generateTestToken("test_secret_key", "User123", []string{"user"})

		c.Request, _ = http.NewRequest("GET", "/", nil)
		c.Request.AddCookie(&http.Cookie{Name: "auth_token", Value: token})

		middleware.AuthMiddleware()(c)

		assert.Equal(t, http.StatusOK, w.Code) // 200 car c.Next() est appelé (le status par défaut est 200)
		assert.False(t, c.IsAborted())

		// Vérification que les variables sont bien dans le contexte
		val, exists := c.Get("userID")
		assert.True(t, exists)
		assert.Equal(t, "User123", val)
	})
}

func TestRequireRole(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Role_Authorized", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		// Simulation : AuthMiddleware a déjà rempli le contexte
		// ATTENTION : jwt-go renvoie des []interface{} pour les tableaux JSON, pas []string
		roles := []interface{}{"admin", "editor"}
		c.Set("roles", roles)

		middleware.RequireRole("admin")(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, c.IsAborted())
	})

	t.Run("Role_Authorized_CaseInsensitive", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		roles := []interface{}{"Admin"} // "Admin" avec majuscule
		c.Set("roles", roles)

		// On demande "admin" minuscule
		middleware.RequireRole("admin")(c)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.False(t, c.IsAborted())
	})

	t.Run("Role_Forbidden", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)

		roles := []interface{}{"user"}
		c.Set("roles", roles)

		middleware.RequireRole("admin")(c)

		assert.Equal(t, http.StatusForbidden, w.Code)
		assert.Contains(t, w.Body.String(), "rôle insuffisant")
		assert.True(t, c.IsAborted())
	})

	t.Run("No_Roles_In_Context", func(t *testing.T) {
		w := httptest.NewRecorder()
		c, _ := gin.CreateTestContext(w)
		// Pas de set("roles")

		middleware.RequireRole("admin")(c)

		assert.Equal(t, http.StatusUnauthorized, w.Code)
		assert.Contains(t, w.Body.String(), "Rôles non définis")
		assert.True(t, c.IsAborted())
	})
}
