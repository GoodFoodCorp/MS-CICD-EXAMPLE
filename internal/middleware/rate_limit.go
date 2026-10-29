package middleware

import (
	"log"
	"os"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
	redisstore "github.com/ulule/limiter/v3/drivers/store/redis"
)

// authRate is the per-client budget on the public auth endpoints (login,
// register, password reset...): enough for a human, too little for a
// credential-stuffing script.
var authRate = limiter.Rate{Period: time.Minute, Limit: 10}

// defaultTrustedProxies are the private ranges the cluster ingress lives in.
// Only a request coming from one of them may set X-Forwarded-For; anything
// else is keyed on its own address, so the header cannot be spoofed to dodge
// the limit.
var defaultTrustedProxies = []string{"10.0.0.0/8", "172.16.0.0/12", "192.168.0.0/16", "127.0.0.1/32", "::1/128"}

// RateLimitMiddleware limits each client IP on the auth endpoints. Counters
// live in Redis when REDIS_URL is set, so the limit holds across replicas;
// otherwise in memory (single instance, local development).
func RateLimitMiddleware() gin.HandlerFunc {
	return mgin.NewMiddleware(limiter.New(newRateStore(os.Getenv("REDIS_URL")), authRate))
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
