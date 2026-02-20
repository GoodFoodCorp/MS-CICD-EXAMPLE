package tests

import (
	"auth-service/internal/middleware"
	"net/http"
	"net/http/httptest"
	"os"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
	"github.com/stretchr/testify/assert"
)

/* Helper pour générer un token valide pour les tests */
func generateTestToken(secret string, userID string, roles []string) string {
	claims := jwt.MapClaims{
		"sub":       userID,
		"tenant_id": "T1",
		"roles":     roles,
		"exp":       time.Now().Add(time.Hour).Unix(),
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	str, _ := token.SignedString([]byte(secret))
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
		assert.Contains(t, w.Body.String(), "Cookie manquant")
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