package tests

import (
	"auth-service/internal/controllers"
	"auth-service/internal/models"
	"bytes"
	"encoding/json"
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
	ctrl := controllers.NewProfileController(new(MockAuthService))

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
	ctrl := controllers.NewProfileController(new(MockAuthService))
	r.GET("/api/user/me", ctrl.GetProfile)
	return r
}

func SetupProfileRouterWithService(service *MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewProfileController(service)

	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "U1")
		c.Next()
	}

	protected := r.Group("/api/user")
	protected.Use(authMiddleware)
	{
		protected.PUT("/me/password", ctrl.ChangePassword)
		protected.DELETE("/me", ctrl.DeleteAccount)
	}

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

// ─── ChangePassword ─────────────────────────────────

func TestProfileController_ChangePassword_Success(t *testing.T) {
	mockService := new(MockAuthService)
	mockService.On("ChangePassword", "U1", &models.ChangePasswordRequest{
		CurrentPassword: "OldPass123", NewPassword: "NewPass456", ConfirmPassword: "NewPass456",
	}).Return(nil)
	router := SetupProfileRouterWithService(mockService)

	body, _ := json.Marshal(map[string]string{
		"current_password": "OldPass123", "new_password": "NewPass456", "confirm_password": "NewPass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/user/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestProfileController_ChangePassword_WrongCurrent(t *testing.T) {
	mockService := new(MockAuthService)
	mockService.On("ChangePassword", "U1", &models.ChangePasswordRequest{
		CurrentPassword: "WrongPass", NewPassword: "NewPass456", ConfirmPassword: "NewPass456",
	}).Return(assert.AnError)
	router := SetupProfileRouterWithService(mockService)

	body, _ := json.Marshal(map[string]string{
		"current_password": "WrongPass", "new_password": "NewPass456", "confirm_password": "NewPass456",
	})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("PUT", "/api/user/me/password", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── DeleteAccount ──────────────────────────────────

func TestProfileController_DeleteAccount_Success(t *testing.T) {
	mockService := new(MockAuthService)
	mockService.On("DeleteAccount", "U1", &models.DeleteAccountRequest{Password: "MyPassword1"}).Return(nil)
	router := SetupProfileRouterWithService(mockService)

	body, _ := json.Marshal(map[string]string{"password": "MyPassword1"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/user/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	mockService.AssertExpectations(t)
}

func TestProfileController_DeleteAccount_WrongPassword(t *testing.T) {
	mockService := new(MockAuthService)
	mockService.On("DeleteAccount", "U1", &models.DeleteAccountRequest{Password: "Wrong"}).Return(assert.AnError)
	router := SetupProfileRouterWithService(mockService)

	body, _ := json.Marshal(map[string]string{"password": "Wrong"})
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("DELETE", "/api/user/me", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}
