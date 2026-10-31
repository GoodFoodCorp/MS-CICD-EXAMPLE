package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

// Budgets on the public auth endpoints. A whole office, campus or mobile
// carrier may share one public address, so the per-address budgets are
// generous; guessing a password is slowed per account instead.
var (
	// AccountRate: logins and reset requests for one email, from anywhere.
	AccountRate = limiter.Rate{Period: time.Minute, Limit: 10}
	// SensitiveRate: login, sign-up, password reset, email check per address.
	SensitiveRate = limiter.Rate{Period: time.Minute, Limit: 60}
	// SessionRate: token refresh, logout, OAuth redirects per address — called
	// by every open tab, never a way to guess anything.
	SessionRate = limiter.Rate{Period: time.Minute, Limit: 600}
)

// defaultTrustedProxies are the private ranges the cluster ingress lives in.
// Only a request coming from one of them may set X-Forwarded-For; anything
// else is keyed on its own address, so the header cannot be spoofed to dodge
// the limit.
var defaultTrustedProxies = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1/32", "::1/128"}

// RateLimits hands out the auth endpoints' limiters. Counters live in Redis
// when REDIS_URL is set, so the limits hold across replicas; otherwise in
// memory (single instance, local development).
type RateLimits struct {
	Account   gin.HandlerFunc
	Sensitive gin.HandlerFunc
	Session   gin.HandlerFunc
}

func NewRateLimits() RateLimits {
	return NewRateLimitsWithStore(newRateStore(os.Getenv("REDIS_URL")))
}

func NewRateLimitsWithStore(store limiter.Store) RateLimits {
	return RateLimits{
		Account:   limit(limiter.New(store, AccountRate), "account", accountKey),
		Sensitive: limit(limiter.New(store, SensitiveRate), "sensitive", addressKey),
		Session:   limit(limiter.New(store, SessionRate), "session", addressKey),
	}
}

func addressKey(c *gin.Context) string { return c.ClientIP() }

// accountKey is the email the request is about, read from its JSON body (put
// back for the handler). No email, no account limit: the handler refuses it.
func accountKey(c *gin.Context) string {
	if c.Request.Body == nil {
		return ""
	}
	raw, err := io.ReadAll(io.LimitReader(c.Request.Body, 64<<10))
	if err != nil {
		return ""
	}
	c.Request.Body = io.NopCloser(bytes.NewReader(raw))
	var body struct {
		Email string `json:"email"`
	}
	if json.Unmarshal(raw, &body) != nil {
		return ""
	}
	return strings.ToLower(strings.TrimSpace(body.Email))
}

func limit(l *limiter.Limiter, name string, key func(*gin.Context) string) gin.HandlerFunc {
	return func(c *gin.Context) {
		k := key(c)
		if k == "" {
			c.Next()
			return
		}
		res, err := l.Get(c.Request.Context(), name+":"+k)
		if err != nil {
			// The counters' store is down: letting people sign in beats
			// locking everyone out; the per-address limits come back with it.
			log.Printf("[WARNING] rate limit %s indisponible: %v", name, err)
			c.Next()
			return
		}
		if res.Reached {
			wait := res.Reset - time.Now().Unix()
			if wait < 1 {
				wait = 1
			}
			c.Header("Retry-After", strconv.FormatInt(wait, 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": fmt.Sprintf("Trop de tentatives. Réessayez dans %d secondes.", wait),
			})
			return
		}
		c.Next()
	}
}

func newRateStore(redisURL string) limiter.Store {
	if redisURL == "" {
		return memory.NewStore()
	}
	opts, err := redis.ParseURL(redisURL)
	if err != nil {
		log.Printf("[WARNING] REDIS_URL invalide (%v) : rate limit en mémoire", err)
		return memory.NewStore()
	}
	store, err := redisstore.NewStoreWithOptions(redis.NewClient(opts), limiter.StoreOptions{Prefix: "auth-rate"})
	if err != nil {
		log.Printf("[WARNING] Redis indisponible (%v) : rate limit en mémoire", err)
		return memory.NewStore()
	}
	return store
}

// TrustedProxies returns the CIDRs allowed to set forwarding headers, from
// TRUSTED_PROXIES (comma separated) or the private-network defaults.
func TrustedProxies() []string {
	raw := strings.TrimSpace(os.Getenv("TRUSTED_PROXIES"))
	if raw == "" {
		return defaultTrustedProxies
	}
	var out []string
	for _, cidr := range strings.Split(raw, ",") {
		if cidr = strings.TrimSpace(cidr); cidr != "" {
			out = append(out, cidr)
		}
	}
	return out
}
