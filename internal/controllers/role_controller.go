package controllers

import (
	"auth-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type RoleController struct {
	service services.RoleService
}

func NewRoleController(service services.RoleService) *RoleController {
	return &RoleController{service: service}
}

// @Summary      Create role
// @Description  Create a new role in tenant
// @Tags         Roles
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body map[string]string true "Role data"
// @Success      201  {object} map[string]string "Role created"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles [post]
func (ctrl *RoleController) CreateRole(c *gin.Context) {
	type RoleReq struct {
		Name        string `json:"name" binding:"required"`
		Description string `json:"description"`
	}
	var req RoleReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.CreateRole(req.Name, req.Description); err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "Rôle créé avec succès"})
}

// @Summary      Assign role to user
// @Description  Assign a role to user
// @Tags         Roles
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body map[string]interface{} true "User ID and Role ID"
// @Success      200  {object} map[string]string "Role assigned"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles/assign [post]
func (ctrl *RoleController) AssignRoleToUser(c *gin.Context) {
	type AssignReq struct {
		UserID string `json:"user_id" binding:"required"`
		RoleID uint   `json:"role_id" binding:"required"`
	}
	var req AssignReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.AssignRoleToUser(req.UserID, req.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rôle assigné avec succès"})
}

// @Summary      Remove role from user
// @Description  Remove a role from user
// @Tags         Roles
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body map[string]interface{} true "User ID and Role ID"
// @Success      200  {object} map[string]string "Role removed"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles/remove [post]
func (ctrl *RoleController) RemoveRoleFromUser(c *gin.Context) {
	type RemoveReq struct {
		UserID string `json:"user_id" binding:"required"`
		RoleID uint   `json:"role_id" binding:"required"`
	}
	var req RemoveReq
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := ctrl.service.RemoveRoleFromUser(req.UserID, req.RoleID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Rôle supprimé avec succès"})
}

// @Summary      Get user roles
// @Description  Get all roles assigned to user
// @Tags         Roles
// @Security     Bearer
// @Produce      json
// @Param        user_id path string true "User ID"
// @Success      200  {object} map[string]interface{} "User roles"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles/user/{user_id} [get]
func (ctrl *RoleController) GetUserRoles(c *gin.Context) {
	userID := c.Param("user_id")

	roles, err := ctrl.service.GetUserRoles(userID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// @Summary      Get all roles
// @Description  Get all roles
// @Tags         Roles
// @Security     Bearer
// @Produce      json
// @Success      200  {object} map[string]interface{} "Roles list"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles [get]
func (ctrl *RoleController) GetAllRoles(c *gin.Context) {
	roles, err := ctrl.service.GetAllRoles()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"roles": roles})
}

// @Summary      Get role by slug
// @Description  Get a role by its slug
// @Tags         Roles
// @Security     Bearer
// @Produce      json
// @Param        slug path string true "Role slug"
// @Success      200  {object} map[string]interface{} "Role"
// @Failure      404  {object} map[string]string "Role not found"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/roles/slug/{slug} [get]
func (ctrl *RoleController) GetRoleBySlug(c *gin.Context) {
	slug := c.Param("slug")

	role, err := ctrl.service.GetRoleBySlug(slug)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Rôle introuvable"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"role": role})
}
