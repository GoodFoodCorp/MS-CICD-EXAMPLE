package middleware

import (
	"fmt"
	"net/http"
	"os"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func AuthMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		tokenString, err := c.Cookie("auth_token")
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Non authentifié (Cookie manquant)"})
			c.Abort()
			return
		}

		token, err := jwt.Parse(tokenString, func(token *jwt.Token) (interface{}, error) {
			if _, ok := token.Method.(*jwt.SigningMethodHMAC); !ok {
				return nil, fmt.Errorf("méthode de signature inattendue: %v", token.Header["alg"])
			}
			return []byte(os.Getenv("JWT_SECRET")), nil
		})

		if err != nil || token == nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
			c.Abort()
			return
		}

		if claims, ok := token.Claims.(jwt.MapClaims); ok && token.Valid {
			c.Set("userID", claims["sub"])
			c.Set("email", claims["email"])
			c.Set("tenantID", claims["tenant_id"])
			c.Set("roles", claims["roles"])
			
			c.Next()
		} else {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Token invalide ou expiré"})
			c.Abort()
		}
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