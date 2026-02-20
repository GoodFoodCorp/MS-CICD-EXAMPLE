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

func SetupRoleRouter(srv *MockRoleService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewRoleController(srv)

	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "U1")
		c.Next()
	}

	admin := r.Group("/api/admin")
	admin.Use(authMiddleware)
	{
		admin.POST("/roles", ctrl.CreateRole)
		admin.POST("/roles/assign", ctrl.AssignRoleToUser)
		admin.POST("/roles/remove", ctrl.RemoveRoleFromUser)
		admin.GET("/roles/user/:user_id", ctrl.GetUserRoles)
		admin.GET("/roles", ctrl.GetAllRoles)
		admin.GET("/roles/slug/:slug", ctrl.GetRoleBySlug)
	}

	return r
}

// ─── CreateRole ─────────────────────────────────────

func TestRoleController_CreateRole_Success(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("CreateRole", "Moderator", "Mod role").Return(nil).Once()

	body := []byte(`{"name": "Moderator", "description": "Mod role"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "créé")
}

func TestRoleController_CreateRole_BadJSON(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles", bytes.NewBufferString(`{bad`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_CreateRole_MissingName(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	body := []byte(`{"description": "Desc"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_CreateRole_Conflict(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("CreateRole", "admin", "").Return(errors.New("ce rôle existe déjà")).Once()

	body := []byte(`{"name": "admin"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── AssignRoleToUser ───────────────────────────────

func TestRoleController_AssignRole_Success(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("AssignRoleToUser", "U1", uint(2)).Return(nil).Once()

	body := []byte(`{"user_id": "U1", "role_id": 2}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "assigné")
}

func TestRoleController_AssignRole_BadJSON(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/assign", bytes.NewBufferString(`{nope`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_AssignRole_MissingFields(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	body := []byte(`{"user_id": "U1"}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_AssignRole_ServiceError(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("AssignRoleToUser", "U1", uint(99)).Return(errors.New("rôle introuvable")).Once()

	body := []byte(`{"user_id": "U1", "role_id": 99}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/assign", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── RemoveRoleFromUser ─────────────────────────────

func TestRoleController_RemoveRole_Success(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("RemoveRoleFromUser", "U1", uint(2)).Return(nil).Once()

	body := []byte(`{"user_id": "U1", "role_id": 2}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/remove", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "supprimé")
}

func TestRoleController_RemoveRole_BadJSON(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/remove", bytes.NewBufferString(`oops`))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRoleController_RemoveRole_ServiceError(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("RemoveRoleFromUser", "U1", uint(2)).Return(errors.New("erreur")).Once()

	body := []byte(`{"user_id": "U1", "role_id": 2}`)
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/admin/roles/remove", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── GetUserRoles ───────────────────────────────────

func TestRoleController_GetUserRoles_Success(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	roles := []models.Role{{ID: 1, Name: "admin"}, {ID: 2, Name: "editor"}}
	mockSrv.On("GetUserRoles", "U1").Return(roles, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/roles/user/U1", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin")
	assert.Contains(t, w.Body.String(), "editor")
}

func TestRoleController_GetUserRoles_ServiceError(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("GetUserRoles", "UNKNOWN").Return([]models.Role{}, errors.New("erreur")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/roles/user/UNKNOWN", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── GetAllRoles ────────────────────────────────────

func TestRoleController_GetAllRoles_Success(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	roles := []models.Role{{ID: 1, Name: "admin"}}
	mockSrv.On("GetAllRoles").Return(roles, nil).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/roles", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "admin")
}

func TestRoleController_GetAllRoles_ServiceError(t *testing.T) {
	mockSrv := new(MockRoleService)
	router := SetupRoleRouter(mockSrv)

	mockSrv.On("GetAllRoles").Return([]models.Role{}, errors.New("db error")).Once()

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/admin/roles", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
