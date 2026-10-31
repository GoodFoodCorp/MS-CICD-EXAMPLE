package tests

import (
	"auth-service/internal/middleware"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func limitsRouter() (*gin.Engine, *string) {
	gin.SetMode(gin.TestMode)
	limits := middleware.NewRateLimitsWithStore(memory.NewStore())
	seen := new(string)
	r := gin.New()
	r.POST("/login", limits.Sensitive, limits.Account, func(c *gin.Context) {
		body, _ := io.ReadAll(c.Request.Body)
		*seen = string(body)
		c.Status(http.StatusOK)
	})
	r.POST("/refresh", limits.Session, func(c *gin.Context) { c.Status(http.StatusOK) })
	return r, seen
}

func post(r *gin.Engine, path, remote, body string) *httptest.ResponseRecorder {
	w := httptest.NewRecorder()
	req, _ := http.NewRequest(http.MethodPost, path, strings.NewReader(body))
	req.RemoteAddr = remote
	r.ServeHTTP(w, req)
	return w
}

// Guessing one account's password is slowed down wherever the attempts come from.
func TestRateLimit_AccountBudgetAcrossAddresses(t *testing.T) {
	r, _ := limitsRouter()
	body := `{"email":"Victim@Example.com","password":"x"}`
	for i := int64(0); i < middleware.AccountRate.Limit; i++ {
		assert.Equal(t, http.StatusOK, post(r, "/login", "198.51.100.1:1", body).Code)
	}
	w := post(r, "/login", "198.51.100.2:1", `{"email":"victim@example.com ","password":"y"}`)
	assert.Equal(t, http.StatusTooManyRequests, w.Code, "same account (case and spaces ignored) from another address")
	assert.NotEmpty(t, w.Header().Get("Retry-After"))
	var answer map[string]string
	assert.NoError(t, json.Unmarshal(w.Body.Bytes(), &answer))
	assert.Contains(t, answer["error"], "Réessayez dans")
	assert.Equal(t, http.StatusOK, post(r, "/login", "198.51.100.2:1", `{"email":"other@example.com"}`).Code)
}

// Many people behind one public address (office, campus, carrier NAT) can
// still sign in: the per-address budget is far above one person's.
func TestRateLimit_SharedAddressKeepsWorking(t *testing.T) {
	r, _ := limitsRouter()
	for i := 0; i < 30; i++ {
		body := `{"email":"user` + string(rune('a'+i%26)) + string(rune('a'+i/26)) + `@example.com"}`
		assert.Equal(t, http.StatusOK, post(r, "/login", "203.0.113.5:1", body).Code)
	}
	for i := 0; i < 200; i++ {
		assert.Equal(t, http.StatusOK, post(r, "/refresh", "203.0.113.5:1", "").Code)
	}
}

// The handler still gets the body the account limit had to read.
func TestRateLimit_BodyIsPassedOn(t *testing.T) {
	r, seen := limitsRouter()
	body := `{"email":"a@b.fr","password":"secret"}`
	post(r, "/login", "192.0.2.1:1", body)
	assert.Equal(t, body, *seen)
}
