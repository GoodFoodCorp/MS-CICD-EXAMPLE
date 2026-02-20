package controllers

import (
	"auth-service/internal/services"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
)

type AdminController struct {
	userAdminService services.UserAdminService
}

func NewAdminController(userAdminSvc services.UserAdminService) *AdminController {
	return &AdminController{userAdminService: userAdminSvc}
}

// @Summary      Get all users
// @Description  List all users with pagination
// @Tags         Admin
// @Security     Bearer
// @Produce      json
// @Param        page query int false "Page number" default(1)
// @Param        limit query int false "Items per page" default(10)
// @Success      200  {object} map[string]interface{} "Users list"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/users [get]
func (ctrl *AdminController) GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	users, total, err := ctrl.userAdminService.GetAllUsers(page, limit)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"data": users, "total": total})
}

// @Summary      Get user by ID
// @Description  Get user details by ID
// @Tags         Admin
// @Security     Bearer
// @Produce      json
// @Param        id path string true "User ID"
// @Success      200  {object} map[string]interface{} "User details"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Failure      404  {object} map[string]string "User not found"
// @Router       /api/admin/users/{id} [get]
func (ctrl *AdminController) GetUserByID(c *gin.Context) {
	u, err := ctrl.userAdminService.GetUserByID(c.Param("id"))
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}
	c.JSON(200, u)
}

// @Summary      Search user by email
// @Description  Get user details by email address
// @Tags         Admin
// @Security     Bearer
// @Produce      json
// @Param        email query string true "User email"
// @Success      200  {object} map[string]interface{} "User details"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Failure      404  {object} map[string]string "User not found"
// @Router       /api/admin/search [get]
func (ctrl *AdminController) GetUserByEmail(c *gin.Context) {
	email := c.Query("email")
	if email == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Email requis"})
		return
	}
	u, err := ctrl.userAdminService.GetUserByEmail(email)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Utilisateur introuvable"})
		return
	}
	c.JSON(200, u)
}

// @Summary      Promote user to admin
// @Description  Assign admin role to user
// @Tags         Admin
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body map[string]string true "User ID"
// @Success      200  {object} map[string]string "User promoted"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/promote [post]
func (ctrl *AdminController) MakeAdmin(c *gin.Context) {
	type Req struct {
		UserID string `json:"user_id" binding:"required"`
	}
	var r Req
	if err := c.ShouldBindJSON(&r); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.userAdminService.PromoteUserToAdmin(r.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Promoted"})
}
