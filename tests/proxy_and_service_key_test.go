package tests

import (
	"auth-service/internal/middleware"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func limitedRouter(t *testing.T) *gin.Engine {
	t.Helper()
	gin.SetMode(gin.TestMode)
	r := gin.New()
	if err := r.SetTrustedProxies(middleware.TrustedProxies()); err != nil {
		t.Fatal(err)
	}
	r.Use(middleware.NewRateLimitsWithStore(memory.NewStore()).Sensitive)
	r.GET("/", func(c *gin.Context) { c.Status(http.StatusOK) })
	return r
}

func hit(r *gin.Engine, remote, forwardedFor string) int {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodGet, "/", nil)
	req.RemoteAddr = remote
	if forwardedFor != "" {
		req.Header.Set("X-Forwarded-For", forwardedFor)
	}
	r.ServeHTTP(w, req)
	return w.Code
}

// Behind the ingress every request comes from the proxy address: clients must
// still be limited individually, from the forwarded address.
func TestRateLimit_KeysOnForwardedClientBehindTrustedProxy(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	r := limitedRouter(t)
	for i := int64(0); i < middleware.SensitiveRate.Limit; i++ {
		assert.Equal(t, http.StatusOK, hit(r, "10.42.0.5:443", "203.0.113.1"))
	}
	assert.Equal(t, http.StatusTooManyRequests, hit(r, "10.42.0.5:443", "203.0.113.1"))
	assert.Equal(t, http.StatusOK, hit(r, "10.42.0.5:443", "203.0.113.2"), "another client keeps its own budget")
}

// A direct client cannot reset its budget by forging X-Forwarded-For.
func TestRateLimit_IgnoresForwardedHeaderFromUntrustedClient(t *testing.T) {
	t.Setenv("TRUSTED_PROXIES", "")
	r := limitedRouter(t)
	for i := int64(0); i < middleware.SensitiveRate.Limit; i++ {
		assert.Equal(t, http.StatusOK, hit(r, "198.51.100.7:5000", "203.0.113.9"))
	}
	assert.Equal(t, http.StatusTooManyRequests, hit(r, "198.51.100.7:5000", "203.0.113.10"))
}

func TestRequireServiceKey(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(middleware.RequireServiceKey())
	r.GET("/internal/users", func(c *gin.Context) { c.Status(http.StatusOK) })
	call := func(key string) int {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest(http.MethodGet, "/internal/users", nil)
		if key != "" {
			req.Header.Set(middleware.ServiceKeyHeader, key)
		}
		r.ServeHTTP(w, req)
		return w.Code
	}

	t.Setenv("SERVICE_API_KEY", "")
	assert.Equal(t, http.StatusUnauthorized, call("anything"), "routes stay closed without a configured key")

	t.Setenv("SERVICE_API_KEY", "s3cret")
	assert.Equal(t, http.StatusUnauthorized, call(""))
	assert.Equal(t, http.StatusUnauthorized, call("wrong"))
	assert.Equal(t, http.StatusOK, call("s3cret"))
}
