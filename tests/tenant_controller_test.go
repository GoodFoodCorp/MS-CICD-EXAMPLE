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
)

// ─── Setup ──────────────────────────────────────────

func SetupTenantRouter(srv *MockTenantService) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	ctrl := controllers.NewTenantController(srv)

	authMiddleware := func(c *gin.Context) {
		c.Set("userID", "U1")
		c.Next()
	}

	admin := r.Group("/api/admin")
	admin.Use(authMiddleware)
	{
		admin.POST("/tenants", ctrl.CreateTenant)
		admin.GET("/tenants", ctrl.GetAllTenants)
		admin.GET("/tenants/:id", ctrl.GetTenantByID)
		admin.GET("/tenants/slug/:slug", ctrl.GetTenantBySlug)
		admin.PUT("/tenants/:id", ctrl.UpdateTenant)
		admin.DELETE("/tenants/:id", ctrl.DeleteTenant)
	}

	return r
}

// ─── CreateTenant ───────────────────────────────────

func TestTenantController_Create_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	req := models.CreateTenantRequest{Name: "My Corp", Plan: "basic"}
	mockSrv.On("CreateTenant", &req).Return(&models.Tenant{ID: "T1", Name: "My Corp", Slug: "my-corp", Plan: "basic"}, nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/admin/tenants", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusCreated, w.Code)
	assert.Contains(t, w.Body.String(), "My Corp")
	assert.Contains(t, w.Body.String(), "créé")
}

func TestTenantController_Create_BadJSON(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/admin/tenants", bytes.NewBufferString(`{bad`))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantController_Create_MissingName(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	body := []byte(`{"plan": "basic"}`)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/admin/tenants", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantController_Create_Conflict(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	req := models.CreateTenantRequest{Name: "Dup Corp", Plan: "basic"}
	mockSrv.On("CreateTenant", &req).Return(nil, errors.New("slug déjà utilisé")).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("POST", "/api/admin/tenants", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusConflict, w.Code)
}

// ─── GetTenantByID ──────────────────────────────────

func TestTenantController_GetByID_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetTenantByID", "T1").Return(&models.Tenant{ID: "T1", Name: "Corp"}, nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants/T1", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Corp")
}

func TestTenantController_GetByID_NotFound(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetTenantByID", "UNKNOWN").Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants/UNKNOWN", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "introuvable")
}

// ─── GetTenantBySlug ────────────────────────────────

func TestTenantController_GetBySlug_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetTenantBySlug", "my-corp").Return(&models.Tenant{ID: "T1", Slug: "my-corp"}, nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants/slug/my-corp", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "my-corp")
}

func TestTenantController_GetBySlug_NotFound(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetTenantBySlug", "nope").Return(nil, errors.New("not found")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants/slug/nope", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusNotFound, w.Code)
	assert.Contains(t, w.Body.String(), "introuvable")
}

// ─── GetAllTenants ──────────────────────────────────

func TestTenantController_GetAll_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	tenants := []models.Tenant{
		{ID: "T1", Name: "Corp A"},
		{ID: "T2", Name: "Corp B"},
	}
	mockSrv.On("GetAllTenants", 1, 10).Return(tenants, int64(2), nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants?page=1&limit=10", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "Corp A")
	assert.Contains(t, w.Body.String(), "Corp B")
}

func TestTenantController_GetAll_DefaultPagination(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetAllTenants", 1, 10).Return([]models.Tenant{}, int64(0), nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTenantController_GetAll_ServiceError(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("GetAllTenants", 1, 10).Return([]models.Tenant{}, int64(0), errors.New("db error")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("GET", "/api/admin/tenants", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── UpdateTenant ───────────────────────────────────

func TestTenantController_Update_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	req := models.UpdateTenantRequest{Name: "Updated Corp"}
	mockSrv.On("UpdateTenant", "T1", &req).Return(nil).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/api/admin/tenants/T1", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "mis à jour")
}

func TestTenantController_Update_BadJSON(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/api/admin/tenants/T1", bytes.NewBufferString(`{bad`))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestTenantController_Update_ServiceError(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	req := models.UpdateTenantRequest{Name: "Fail"}
	mockSrv.On("UpdateTenant", "T1", &req).Return(errors.New("erreur interne")).Once()

	body, _ := json.Marshal(req)
	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("PUT", "/api/admin/tenants/T1", bytes.NewBuffer(body))
	httpReq.Header.Set("Content-Type", "application/json")
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}

// ─── DeleteTenant ───────────────────────────────────

func TestTenantController_Delete_Success(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("DeleteTenant", "T1").Return(nil).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("DELETE", "/api/admin/tenants/T1", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "supprimé")
}

func TestTenantController_Delete_ServiceError(t *testing.T) {
	mockSrv := new(MockTenantService)
	router := SetupTenantRouter(mockSrv)

	mockSrv.On("DeleteTenant", "T1").Return(errors.New("erreur")).Once()

	w := httptest.NewRecorder()
	httpReq, _ := http.NewRequest("DELETE", "/api/admin/tenants/T1", nil)
	router.ServeHTTP(w, httpReq)

	assert.Equal(t, http.StatusInternalServerError, w.Code)
}
