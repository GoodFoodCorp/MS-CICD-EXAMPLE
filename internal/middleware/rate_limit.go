package middleware

import (
	"time"

	"github.com/gin-gonic/gin"
	"github.com/ulule/limiter/v3"
	mgin "github.com/ulule/limiter/v3/drivers/middleware/gin"
	"github.com/ulule/limiter/v3/drivers/store/memory"
)

func RateLimitMiddleware() gin.HandlerFunc {
	store := memory.NewStore()

	instance := limiter.New(store, limiter.Rate{
		Period: 1 * time.Minute,
		Limit:  10,
	})

	middleware := mgin.NewMiddleware(instance)

	return middleware
}
