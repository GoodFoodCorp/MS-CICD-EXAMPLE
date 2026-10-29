package middleware

import (
	"net/http"
	"strings"

	"auth-service/internal/tokens"

	"github.com/gin-gonic/gin"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Cookie first (web), Authorization: Bearer as fallback (mobile / services)
		tokenString, err := c.Cookie("auth_token")
		if err != nil || tokenString == "" {
			authHeader := c.GetHeader("Authorization")
			if strings.HasPrefix(authHeader, "Bearer ") {
				tokenString = strings.TrimPrefix(authHeader, "Bearer ")
			}
		}
		if tokenString == "" {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié (token manquant)"})
			c.Abort()
			return
		}

		cfg, err := tokens.FromEnv()
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Configuration JWT absente"})
			c.Abort()
			return
		}
		claims, err := cfg.ParseAccess(tokenString)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
			c.Abort()
			return
		}
		c.Set("userID", claims["sub"])
		c.Set("email", claims["email"])
		c.Set("tenantID", claims["tenant_id"])
		c.Set("roles", claims["roles"])
		c.Next()
	}
}

func RequireRole(requiredRole string) gin.HandlerFunc {
	return func(c *gin.Context) {
		rolesInterface, exists := c.Get("roles")
		if !exists {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Rôles non définis"})
			c.Abort()
			return
		}

		// Convertit l'interface en slice de strings
		roles, ok := rolesInterface.([]interface{})
		if !ok {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Format de rôles invalide"})
			c.Abort()
			return
		}

		// Vérifie si l'utilisateur a le rôle requis (case-insensitive)
		requiredRoleLower := strings.ToLower(requiredRole)
		hasRole := false
		for _, role := range roles {
			if roleStr, ok := role.(string); ok && strings.ToLower(roleStr) == requiredRoleLower {
				hasRole = true
				break
			}
		}

		if !hasRole {
			c.JSON(http.StatusForbidden, gin.H{"error": "Accès refusé: rôle insuffisant"})
			c.Abort()
			return
		}

		c.Next()
	}
}
