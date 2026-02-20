package controllers

import (
	"github.com/gin-gonic/gin"
)

type ProfileController struct{}

func NewProfileController() *ProfileController {
	return &ProfileController{}
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
