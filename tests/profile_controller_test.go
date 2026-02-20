package tests

import (
	"auth-service/internal/controllers"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// ─── Setup ──────────────────────────────────────────

func SetupProfileRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewProfileController()

	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "U1")
		c.Next()
	}

	protected := r.Group("/api/user")
	protected.Use(authMiddleware)
	{
		protected.GET("/me", ctrl.GetProfile)
	}

	return r
}

func SetupProfileRouterNoAuth() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewProfileController()
	r.GET("/api/user/me", ctrl.GetProfile)
	return r
}

// ─── GetProfile ─────────────────────────────────────

func TestProfileController_GetProfile_Success(t *testing.T) {
	router := SetupProfileRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/user/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "U1")
	assert.Contains(t, w.Body.String(), "user_id")
}

func TestProfileController_GetProfile_NoUserID(t *testing.T) {
	router := SetupProfileRouterNoAuth()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/user/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user_id")
}

func TestProfileController_GetProfile_WrongMethod(t *testing.T) {
	router := SetupProfileRouter()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/user/me", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}
