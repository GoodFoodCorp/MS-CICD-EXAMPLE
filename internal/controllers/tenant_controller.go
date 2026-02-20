package controllers

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type TenantController struct {
	service services.TenantService
}

func NewTenantController(service services.TenantService) *TenantController {
	return &TenantController{service: service}
}

// @Summary      Create tenant
// @Description  Create a new tenant (slug is auto-generated from name if not provided)
// @Tags         Tenants
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body models.CreateTenantRequest true "Tenant data"
// @Success      201  {object} map[string]interface{} "Tenant created"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/tenants [post]
func (ctrl *TenantController) CreateTenant(c *gin.Context) {
	var req models.CreateTenantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant, err := ctrl.service.CreateTenant(&req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"data": tenant, "message": "Tenant créé avec succès"})
}

// @Summary      Get tenant by ID
// @Description  Get tenant details by ID
// @Tags         Tenants
// @Security     Bearer
// @Produce      json
// @Param        id path string true "Tenant ID"
// @Success      200  {object} map[string]interface{} "Tenant details"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Failure      404  {object} map[string]string "Tenant not found"
// @Router       /api/admin/tenants/{id} [get]
func (ctrl *TenantController) GetTenantByID(c *gin.Context) {
	id := c.Param("id")

	tenant, err := ctrl.service.GetTenantByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant introuvable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tenant})
}

// @Summary      Get tenant by slug
// @Description  Get tenant details by slug
// @Tags         Tenants
// @Security     Bearer
// @Produce      json
// @Param        slug path string true "Tenant slug"
// @Success      200  {object} map[string]interface{} "Tenant details"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Failure      404  {object} map[string]string "Tenant not found"
// @Router       /api/admin/tenants/slug/{slug} [get]
func (ctrl *TenantController) GetTenantBySlug(c *gin.Context) {
	slug := c.Param("slug")

	tenant, err := ctrl.service.GetTenantBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant introuvable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tenant})
}

// @Summary      Get all tenants
// @Description  List all tenants with pagination
// @Tags         Tenants
// @Security     Bearer
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Success      200  {object} map[string]interface{} "Tenants list"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/tenants [get]
func (ctrl *TenantController) GetAllTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	tenants, total, err := ctrl.service.GetAllTenants(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"data": tenants, "total": total})
}

// @Summary      Update tenant
// @Description  Update tenant details
// @Tags         Tenants
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id path string true "Tenant ID"
// @Param        body body models.UpdateTenantRequest true "Tenant data"
// @Success      200  {object} map[string]string "Tenant updated"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/tenants/{id} [put]
func (ctrl *TenantController) UpdateTenant(c *gin.Context) {
	id := c.Param("id")
	var req models.UpdateTenantRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.UpdateTenant(id, &req); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant mis à jour avec succès"})
}

// @Summary      Delete tenant
// @Description  Delete a tenant
// @Tags         Tenants
// @Security     Bearer
// @Produce      json
// @Param        id path string true "Tenant ID"
// @Success      200  {object} map[string]string "Tenant deleted"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Failure      404  {object} map[string]string "Tenant not found"
// @Router       /api/admin/tenants/{id} [delete]
func (ctrl *TenantController) DeleteTenant(c *gin.Context) {
	id := c.Param("id")

	if err := ctrl.service.DeleteTenant(id); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Tenant supprimé avec succès"})
}
