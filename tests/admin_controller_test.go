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

// ─── Setup ──────────────────────────────────────────

func SetupAdminRouter(srv *MockUserAdminService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewAdminController(srv)

	// Middleware fake d'authentification
	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "U1")
		c.Next()
	}

	admin := r.Group("/api/admin")
	admin.Use(authMiddleware)
	{
		admin.GET("/users", ctrl.GetAllUsers)
		admin.GET("/users/:id", ctrl.GetUserByID)
		admin.GET("/search", ctrl.GetUserByEmail)
		admin.POST("/promote", ctrl.MakeAdmin)
	}

	return r
}

// ─── GetAllUsers ────────────────────────────────────

func TestAdminController_GetAllUsers_Success(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	users := []models.User{{ID: "U1", Email: "a@b.com"}, {ID: "U2", Email: "c@d.com"}}
	mockSrv.On("GetAllUsers", 1, 10).Return(users, int64(2), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/users?page=1&limit=10", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "a@b.com")
	assert.Contains(t, w.Body.String(), "c@d.com")
}

func TestAdminController_GetAllUsers_DefaultPagination(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetAllUsers", 1, 10).Return([]models.User{}, int64(0), nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdminController_GetAllUsers_ServiceError(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetAllUsers", 1, 10).Return([]models.User{}, int64(0), errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/users", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── GetUserByID ────────────────────────────────────

func TestAdminController_GetUserByID_Success(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetUserByID", "U1").Return(&models.User{ID: "U1", Email: "user@test.com"}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/users/U1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "user@test.com")
}

func TestAdminController_GetUserByID_NotFound(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetUserByID", "UNKNOWN").Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/users/UNKNOWN", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "introuvable")
}

// ─── GetUserByEmail ─────────────────────────────────

func TestAdminController_GetUserByEmail_Success(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetUserByEmail", "e@t.com").Return(&models.User{Email: "e@t.com"}, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/search?email=e@t.com", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "e@t.com")
}

func TestAdminController_GetUserByEmail_MissingParam(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/search", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
	assert.Contains(t, w.Body.String(), "Email requis")
}

func TestAdminController_GetUserByEmail_NotFound(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("GetUserByEmail", "x@y.com").Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/search?email=x@y.com", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "introuvable")
}

// ─── MakeAdmin (Promote) ───────────────────────────

func TestAdminController_MakeAdmin_Success(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("PromoteUserToAdmin", "U1").Return(nil).Once()

	body := []byte(`{"user_id": "U1"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/promote", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Promoted")
}

func TestAdminController_MakeAdmin_BadJSON(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/promote", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminController_MakeAdmin_MissingUserID(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	body := []byte(`{}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/promote", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestAdminController_MakeAdmin_ServiceError(t *testing.T) {
	mockSrv := new(MockUserAdminService)
	router := SetupAdminRouter(mockSrv)

	mockSrv.On("PromoteUserToAdmin", "U1").Return(errors.New("erreur interne")).Once()

	body := []byte(`{"user_id": "U1"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/promote", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
