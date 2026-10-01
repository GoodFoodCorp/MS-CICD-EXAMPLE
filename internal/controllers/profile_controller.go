package controllers

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type ProfileController struct {
	service services.AuthService
}

func NewProfileController(service services.AuthService) *ProfileController {
	return &ProfileController{service: service}
}

// @Summary      Get user profile
// @Description  Get authenticated user profile with roles
// @Tags         Auth
// @Security     Bearer
// @Produce      json
// @Success      200  {object} map[string]interface{} "User profile"
// @Failure      401  {object} map[string]string "Unauthorized"
// @Router       /api/user/me [get]
func (ctrl *ProfileController) GetProfile(c *gin.Context) {
	roles, _ := c.Get("roles")
	c.JSON(200, gin.H{
		"user_id":   c.GetString("userID"),
		"email":     c.GetString("email"),
		"tenant_id": c.GetString("tenantID"),
		"roles":     roles,
	})
}

// @Summary      Change password
// @Description  Change the authenticated user's password (requires the current one)
// @Tags         Auth
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body models.ChangePasswordRequest true "Password change data"
// @Success      200  {object} map[string]string "Password changed"
// @Failure      400  {object} map[string]string "Invalid input or wrong current password"
// @Router       /api/user/me/password [put]
func (ctrl *ProfileController) ChangePassword(c *gin.Context) {
	var req models.ChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.service.ChangePassword(c.GetString("userID"), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mot de passe modifié avec succès"})
}

// @Summary      Delete account
// @Description  Permanently delete the authenticated user's own account (requires password)
// @Tags         Auth
// @Security     Bearer
// @Accept       json
// @Produce      json
// @Param        body body models.DeleteAccountRequest true "Password confirmation"
// @Success      200  {object} map[string]string "Account deleted"
// @Failure      400  {object} map[string]string "Wrong password"
// @Router       /api/user/me [delete]
func (ctrl *ProfileController) DeleteAccount(c *gin.Context) {
	var req models.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.service.DeleteAccount(c.GetString("userID"), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Compte supprimé"})
}
