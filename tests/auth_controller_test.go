package tests

import (
	"auth-service/internal/controllers"
	"auth-service/internal/models"
	"bytes"
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
)

// ─── Setup ──────────────────────────────────────────

func SetupAuthRouter(srv *MockAuthService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewAuthController(srv)

	r.POST("/api/auth/register", ctrl.Register)
	r.POST("/api/auth/login", ctrl.Login)
	r.GET("/api/auth/verify-email", ctrl.VerifyEmail)
	r.POST("/api/auth/forgot-password", ctrl.ForgotPassword)
	r.POST("/api/auth/reset-password", ctrl.ResetPassword)
	r.POST("/api/auth/refresh", ctrl.Refresh)
	r.POST("/api/auth/logout", ctrl.Logout)

	return r
}

// ─── Register ───────────────────────────────────────

func TestAuthController_Register_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.RegisterRequest{Email: "test@test.com", Password: "StrongPassword123!", TenantID: "T1"}
	mockSrv.On("RegisterUser", &req).Return("user-id-123", nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "user_id")
}

func TestAuthController_Register_Success_NoTenant(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.RegisterRequest{Email: "customer@test.com", Password: "StrongPassword123!"}
	mockSrv.On("RegisterUser", &req).Return("user-id-456", nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "user_id")
}

func TestAuthController_Register_BadJSON(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBufferString(`{bad json`))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Register_MissingFields(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	body, _ := json.Marshal(map[string]string{"email": "test@test.com"})
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Register_Conflict(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.RegisterRequest{Email: "dup@test.com", Password: "StrongPassword123!", TenantID: "T1"}
	mockSrv.On("RegisterUser", &req).Return("", errors.New("cet email est déjà utilisé")).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/register", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── Login ──────────────────────────────────────────

func TestAuthController_Login_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.LoginRequest{Email: "l@t.com", Password: "StrongPassword123!"}
	mockSrv.On("LoginUser", &req).Return("at", "rt", nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Result().Cookies())

	// Vérifier les cookies auth_token et refresh_token
	cookies := w.Result().Cookies()
	var hasAuth, hasRefresh bool
	for _, c := range cookies {
		if c.Name == "auth_token" {
			hasAuth = true
		}
		if c.Name == "refresh_token" {
			hasRefresh = true
		}
	}
	assert.True(t, hasAuth, "Cookie auth_token attendu")
	assert.True(t, hasRefresh, "Cookie refresh_token attendu")
}

func TestAuthController_Login_BadJSON(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBufferString(`not json`))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_Login_Unauthorized(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.LoginRequest{Email: "wrong@t.com", Password: "WrongPass123!"}
	mockSrv.On("LoginUser", &req).Return("", "", errors.New("identifiants incorrects")).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/login", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── VerifyEmail ────────────────────────────────────

func TestAuthController_VerifyEmail_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("VerifyEmail", "valid-tok").Return(nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/auth/verify-email?token=valid-tok", nil)
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "vérifié")
}

func TestAuthController_VerifyEmail_MissingToken(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/auth/verify-email", nil)
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Token manquant")
}

func TestAuthController_VerifyEmail_InvalidToken(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("VerifyEmail", "bad-tok").Return(errors.New("token invalide")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/auth/verify-email?token=bad-tok", nil)
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── Refresh ────────────────────────────────────────

func TestAuthController_Refresh_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("RefreshAccessToken", "old_rt").Return("new_at", "new_rt", nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	httpReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: "old_rt"})
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Refreshed")
}

func TestAuthController_Refresh_NoCookie(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestAuthController_Refresh_Expired(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("RefreshAccessToken", "expired_rt").Return("", "", errors.New("expired")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/refresh", nil)
	httpReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: "expired_rt"})
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

// ─── Logout ─────────────────────────────────────────

func TestAuthController_Logout_WithCookie(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("LogoutUser", "rt_cookie").Return(nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	httpReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt_cookie"})
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Logged out")
}

func TestAuthController_Logout_NoCookie(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/logout", nil)
	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
}

// ─── ForgotPassword ─────────────────────────────────

func TestAuthController_ForgotPassword_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("ForgotPassword", mock.MatchedBy(func(r *models.ForgotPasswordRequest) bool {
		return r.Email == "admin@test.com"
	})).Return("dev_tok", nil).Once()

	req := models.ForgotPasswordRequest{Email: "admin@test.com"}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/forgot-password", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "dev_tok")
}

func TestAuthController_ForgotPassword_BadJSON(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/forgot-password", bytes.NewBufferString(`{bad`))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_ForgotPassword_ServiceError(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	mockSrv.On("ForgotPassword", mock.Anything).Return("", errors.New("utilisateur introuvable")).Once()

	req := models.ForgotPasswordRequest{Email: "unknown@test.com"}
	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/forgot-password", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

// ─── ResetPassword ──────────────────────────────────

func TestAuthController_ResetPassword_Success(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.ResetPasswordRequest{Token: "tok", NewPassword: "StrongPassword123!", ConfirmPassword: "StrongPassword123!"}
	mockSrv.On("ResetPassword", &req).Return(nil).Once()
	mockSrv.On("LogoutUser", "rt").Return(nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/reset-password", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.AddCookie(&http.Cookie{Name: "refresh_token", Value: "rt"})

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "réinitialisé")
}

func TestAuthController_ResetPassword_BadJSON(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/reset-password", bytes.NewBufferString(`nope`))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_ResetPassword_ServiceError(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.ResetPasswordRequest{Token: "bad", NewPassword: "StrongPassword123!", ConfirmPassword: "StrongPassword123!"}
	mockSrv.On("ResetPassword", &req).Return(errors.New("token expiré")).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/reset-password", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAuthController_ResetPassword_NoCookie(t *testing.T) {
	mockSrv := new(MockAuthService)
	router := SetupAuthRouter(mockSrv)

	req := models.ResetPasswordRequest{Token: "tok", NewPassword: "StrongPassword123!", ConfirmPassword: "StrongPassword123!"}
	mockSrv.On("ResetPassword", &req).Return(nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/auth/reset-password", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")

	router.ServeHTTP(w, httpReq)
	assert.Equal(t, http.StatusOK, w.Code)
}
