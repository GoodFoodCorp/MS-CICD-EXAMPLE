package tests

import (
	"auth-service/internal/controllers"
	"auth-service/internal/models"
	"bytes"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

// setupMobileAuthRouter mounts the mobile account routes behind a fake auth
// step that injects the authenticated user, as AuthMiddleware does.
func setupMobileAuthRouter(srv *MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) { c.Set("userID", "U1"); c.Next() })
	ctrl := controllers.NewAuthController(srv)
	r.POST("/api/auth/upgrade-to-delivery", ctrl.UpgradeToDelivery)
	r.POST("/api/auth/change-password", ctrl.ChangePassword)
	r.DELETE("/api/auth/account", ctrl.DeleteAccount)
	return r
}

func doJSON(r *gin.Engine, method, path, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(method, path, bytes.NewBufferString(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)
	return w
}

func TestAuthController_UpgradeToDelivery(t *testing.T) {
	srv := new(MockAuthService)
	srv.On("UpgradeToDelivery", "U1").Return(nil).Once()

	w := doJSON(setupMobileAuthRouter(srv), http.MethodPost, "/api/auth/upgrade-to-delivery", "")

	assert.Equal(t, http.StatusOK, w.Code)
	srv.AssertExpectations(t)
}

func TestAuthController_ChangePassword_ConfirmsWithNewPassword(t *testing.T) {
	srv := new(MockAuthService)
	expected := &models.ChangePasswordRequest{CurrentPassword: "OldPass123", NewPassword: "NewPass123", ConfirmPassword: "NewPass123"}
	srv.On("ChangePassword", "U1", expected).Return(nil).Once()

	w := doJSON(setupMobileAuthRouter(srv), http.MethodPost, "/api/auth/change-password",
		`{"current_password":"OldPass123","new_password":"NewPass123"}`)

	assert.Equal(t, http.StatusOK, w.Code)
	srv.AssertExpectations(t)
}

func TestAuthController_ChangePassword_MissingField(t *testing.T) {
	w := doJSON(setupMobileAuthRouter(new(MockAuthService)), http.MethodPost, "/api/auth/change-password",
		`{"current_password":"OldPass123"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_DeleteAccount_RequiresPassword(t *testing.T) {
	w := doJSON(setupMobileAuthRouter(new(MockAuthService)), http.MethodDelete, "/api/auth/account", `{}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_DeleteAccount_WrongPassword(t *testing.T) {
	srv := new(MockAuthService)
	srv.On("DeleteAccount", "U1", &models.DeleteAccountRequest{Password: "bad"}).
		Return(errors.New("mot de passe incorrect")).Once()

	w := doJSON(setupMobileAuthRouter(srv), http.MethodDelete, "/api/auth/account", `{"password":"bad"}`)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	srv.AssertExpectations(t)
}
