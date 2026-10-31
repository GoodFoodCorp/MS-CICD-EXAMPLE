package controllers

import (
	"auth-service/internal/services"
	"errors"
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
// @Param        limit query int false "Items per page (max 100)" default(10)
// @Param        q query string false "Email contains"
// @Success      200  {object} map[string]interface{} "Users list"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/admin/users [get]
func (ctrl *AdminController) GetAllUsers(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))
	users, total, err := ctrl.userAdminService.GetAllUsers(page, limit, c.Query("q"))
	if err != nil {
		internalError(c, err)
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
		c.JSON(http.StatusBadRequest, gin.H{"error": bindingMessage(err)})
		return
	}
	if err := ctrl.userAdminService.PromoteUserToAdmin(r.UserID); err != nil {
		internalError(c, err)
		return
	}
	c.JSON(200, gin.H{"message": "Promoted"})
}

// @Summary      Set a user's role
// @Description  Gives a user one role (user, manager, livreur, admin); a manager is attached to a restaurant. Their sessions are revoked so the change applies at once.
// @Tags         Admin
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        id path string true "User ID"
// @Param        body body map[string]string true "role, restaurant_id"
// @Success      200  {object} models.User
// @Failure      400  {object} map[string]string "Unknown role or missing restaurant"
// @Failure      403  {object} map[string]string "Self-demotion"
// @Failure      404  {object} map[string]string "User not found"
// @Router       /api/admin/users/{id}/role [put]
func (ctrl *AdminController) SetUserAccess(c *gin.Context) {
	var body struct {
		Role         string `json:"role" binding:"required"`
		RestaurantID string `json:"restaurant_id"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "le rôle est obligatoire"})
		return
	}
	user, err := ctrl.userAdminService.SetUserAccess(c.GetString("userID"), c.Param("id"), body.Role, body.RestaurantID)
	switch {
	case errors.Is(err, services.ErrUnknownRole), errors.Is(err, services.ErrManagerNeedsShop):
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrSelfDemotion):
		c.JSON(http.StatusForbidden, gin.H{"error": err.Error()})
	case errors.Is(err, services.ErrUserNotFound):
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, gin.H{"error": "mise à jour impossible"})
	default:
		c.JSON(http.StatusOK, user)
	}
}
