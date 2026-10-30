package controllers

import (
	"auth-service/internal/models"
	"auth-service/internal/services"
	"net/http"

	"github.com/gin-gonic/gin"
)

type AuthController struct {
	service services.AuthService
}

// UpgradeToDelivery grants the courier role to the authenticated user (mobile
// courier sign-up).
func (ctrl *AuthController) UpgradeToDelivery(c *gin.Context) {
	if err := ctrl.service.UpgradeToDelivery(c.GetString("userID")); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Compte livreur activé"})
}

// ChangePassword is the mobile flavour of PUT /api/user/me/password: the app
// does not send a confirmation field, so the new password confirms itself.
func (ctrl *AuthController) ChangePassword(c *gin.Context) {
	var body struct {
		CurrentPassword string `json:"current_password" binding:"required"`
		NewPassword     string `json:"new_password" binding:"required,min=8"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Les deux mots de passe sont obligatoires"})
		return
	}
	req := &models.ChangePasswordRequest{
		CurrentPassword: body.CurrentPassword,
		NewPassword:     body.NewPassword,
		ConfirmPassword: body.NewPassword,
	}
	if err := ctrl.service.ChangePassword(c.GetString("userID"), req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Mot de passe modifié"})
}

// DeleteAccount deletes the caller's account. The password is required: a
// stolen session alone must never be enough to destroy an account.
func (ctrl *AuthController) DeleteAccount(c *gin.Context) {
	var req models.DeleteAccountRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Le mot de passe est obligatoire"})
		return
	}
	if err := ctrl.service.DeleteAccount(c.GetString("userID"), &req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(http.StatusOK, gin.H{"message": "Compte supprimé"})
}

func NewAuthController(service services.AuthService) *AuthController {
	return &AuthController{service: service}
}

// @Summary      Verify email
// @Description  Verify user email address
// @Tags         Auth
// @Produce      json
// @Param        token query string true "Verification token"
// @Success      200  {object} map[string]string "Email verified"
// @Failure      400  {object} map[string]string "Invalid token"
// @Router       /api/auth/verify-email [get]
func (ctrl *AuthController) VerifyEmail(c *gin.Context) {
	token := c.Query("token")
	if token == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Token manquant"})
		return
	}

	if err := ctrl.service.VerifyEmail(token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Email vérifié avec succès. Vous pouvez vous connecter."})
}

// @Summary      Register user
// @Description  Create a new user account
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body models.RegisterRequest true "Registration data"
// @Success      201  {object} map[string]string "Account created successfully"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      409  {object} map[string]string "Email already in use"
// @Router       /api/auth/register [post]
func (ctrl *AuthController) Register(c *gin.Context) {
	var req models.RegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	userID, err := ctrl.service.RegisterUser(&req)
	if err != nil {
		c.JSON(http.StatusConflict, gin.H{"error": err.Error()})
		return
	}
	// The client signs in at once, or tells the user to open the email first.
	verify := services.EmailVerificationRequired()
	message := "Compte créé."
	if verify {
		message = "Compte créé. Confirmez votre adresse email pour vous connecter."
	}
	c.JSON(http.StatusCreated, gin.H{"message": message, "user_id": userID, "email_verification_required": verify})
}

// @Summary      Login user
// @Description  Authenticate user and return tokens
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body models.LoginRequest true "Login credentials"
// @Success      200  {object} map[string]string "Authentication successful"
// @Failure      400  {object} map[string]string "Invalid input"
// @Failure      401  {object} map[string]string "Incorrect credentials"
// @Router       /api/auth/login [post]
func (ctrl *AuthController) Login(c *gin.Context) {
	var req models.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	accessToken, refreshToken, err := ctrl.service.LoginUser(&req)
	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": err.Error()})
		return
	}
	setSessionCookies(c, accessToken, refreshToken)
	// Tokens also returned in the body so non-browser clients (mobile app,
	// WebSocket handshake) can use the Authorization: Bearer scheme.
	c.JSON(http.StatusOK, gin.H{
		"message":       "Authentification réussie",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// @Summary      Refresh token
// @Description  Refresh access token using refresh token
// @Tags         Auth
// @Produce      json
// @Success      200  {object} map[string]string "Token refreshed"
// @Failure      401  {object} map[string]string "Token expired"
// @Router       /api/auth/refresh [post]
func (ctrl *AuthController) Refresh(c *gin.Context) {
	// Cookie first (web), JSON body as fallback (mobile / non-browser clients)
	old, err := c.Cookie("refresh_token")
	if err != nil || old == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if bindErr := c.ShouldBindJSON(&body); bindErr == nil && body.RefreshToken != "" {
			old = body.RefreshToken
		}
	}
	if old == "" {
		c.JSON(401, gin.H{"error": "No token"})
		return
	}
	newA, newR, err := ctrl.service.RefreshAccessToken(old)
	if err != nil {
		c.JSON(401, gin.H{"error": "Expired"})
		return
	}
	setSessionCookies(c, newA, newR)
	c.JSON(200, gin.H{
		"message":       "Refreshed",
		"access_token":  newA,
		"refresh_token": newR,
		"token_type":    "Bearer",
		"expires_in":    900,
	})
}

// @Summary      Logout user
// @Description  Logout user and revoke tokens
// @Tags         Auth
// @Produce      json
// @Success      200  {object} map[string]string "Logout successful"
// @Router       /api/auth/logout [post]
func (ctrl *AuthController) Logout(c *gin.Context) {
	// Cookie first (web), JSON body as fallback (mobile / non-browser clients)
	rt, _ := c.Cookie("refresh_token")
	if rt == "" {
		var body struct {
			RefreshToken string `json:"refresh_token"`
		}
		if bindErr := c.ShouldBindJSON(&body); bindErr == nil {
			rt = body.RefreshToken
		}
	}
	if rt != "" {
		ctrl.service.LogoutUser(rt)
	}
	clearSessionCookies(c)
	c.JSON(200, gin.H{"message": "Logged out"})
}

// @Summary      Forgot password
// @Description  Request password reset link
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body models.ForgotPasswordRequest true "Email"
// @Success      200  {object} map[string]string "Reset link sent"
// @Failure      400  {object} map[string]string "Invalid input"
// @Router       /api/auth/forgot-password [post]
func (ctrl *AuthController) ForgotPassword(c *gin.Context) {
	var req models.ForgotPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	if _, err := ctrl.service.ForgotPassword(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}
	c.JSON(200, gin.H{"message": "Email envoyé"})
}

// @Summary      Reset password
// @Description  Reset password with valid token
// @Tags         Auth
// @Accept       json
// @Produce      json
// @Param        body body models.ResetPasswordRequest true "Reset data"
// @Success      200  {object} map[string]string "Password reset successfully"
// @Failure      400  {object} map[string]string "Invalid token or passwords"
// @Router       /api/auth/reset-password [post]
func (ctrl *AuthController) ResetPassword(c *gin.Context) {
	var req models.ResetPasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}
	if err := ctrl.service.ResetPassword(&req); err != nil {
		c.JSON(400, gin.H{"error": err.Error()})
		return
	}

	rt, _ := c.Cookie("refresh_token")
	if rt != "" {
		ctrl.service.LogoutUser(rt)
	}

	clearSessionCookies(c)

	c.JSON(200, gin.H{"message": "Votre mot de passe a été réinitialisé avec succès."})
}
