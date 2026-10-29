package middleware

import (
	"crypto/sha256"
	"crypto/subtle"
	"net/http"
	"os"

	"github.com/gin-gonic/gin"
)

// ServiceKeyHeader carries the shared secret of service-to-service calls.
const ServiceKeyHeader = "X-Service-Key"

// RequireServiceKey guards /internal routes: the caller must present
// SERVICE_API_KEY. Without a configured key the routes stay closed.
func RequireServiceKey() gin.HandlerFunc {
	return func(c *gin.Context) {
		expected := os.Getenv("SERVICE_API_KEY")
		provided := c.GetHeader(ServiceKeyHeader)
		if expected == "" || provided == "" || !sameSecret(expected, provided) {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{"error": "Clé de service invalide ou absente"})
			return
		}
		c.Next()
	}
}

// sameSecret compares in constant time whatever the lengths.
func sameSecret(a, b string) bool {
	ha, hb := sha256.Sum256([]byte(a)), sha256.Sum256([]byte(b))
	return subtle.ConstantTimeCompare(ha[:], hb[:]) == 1
}
