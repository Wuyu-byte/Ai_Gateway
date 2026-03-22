package middleware

import (
	"context"
	"fmt"
	"net/http"
	"time"

	"ai-gateway/model"

	"github.com/gin-gonic/gin"
	"github.com/redis/go-redis/v9"
)

func RateLimitMiddleware(redisClient *redis.Client, defaultLimit int) gin.HandlerFunc {
	return func(c *gin.Context) {
		apiKeyValue, ok := c.Get(ContextAPIKeyKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "api key context is missing",
			})
			return
		}

		apiKey, ok := apiKeyValue.(*model.APIKey)
		if !ok {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error": "api key context is invalid",
			})
			return
		}

		limit := defaultLimit
		if apiKey.RateLimit > 0 {
			limit = apiKey.RateLimit
		}

		ctx := context.Background()
		cacheKey := fmt.Sprintf("rate_limit:%s", apiKey.KeyHash)
		count, err := redisClient.Incr(ctx, cacheKey).Result()
		if err != nil {
			c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
				"error": "rate limit service unavailable",
			})
			return
		}

		if count == 1 {
			_ = redisClient.Expire(ctx, cacheKey, time.Minute).Err()
		}

		if count > int64(limit) {
			ttl, _ := redisClient.TTL(ctx, cacheKey).Result()
			c.Header("Retry-After", fmt.Sprintf("%d", int(ttl.Seconds())))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error": "rate limit exceeded",
			})
			return
		}

		c.Next()
	}
}
