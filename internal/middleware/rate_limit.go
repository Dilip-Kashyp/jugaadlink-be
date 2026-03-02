package middleware

import (
	"fmt"
	"net/http"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
)

func RateLimiter() gin.HandlerFunc {
	return func(c *gin.Context) {
		if config.RedisClient == nil {
			c.Next()
			return
		}

		var identifier string
		limit := 5 // Default for guests
		window := time.Minute

		if userID, ok := c.Get("user_id"); ok {
			identifier = fmt.Sprintf("user:%v", userID)
			limit = 50 // Higher limit for registered users
		} else {
			identifier = fmt.Sprintf("ip:%s", c.ClientIP())
		}

		key := fmt.Sprintf("rate_limit:%s", identifier)
		ctx := c.Request.Context()

		count, err := config.RedisClient.Incr(ctx, key).Result()
		if err != nil {
			c.Next()
			return
		}

		if count == 1 {
			config.RedisClient.Expire(ctx, key, window)
		}

		if count > int64(limit) {
			c.AbortWithStatusJSON(http.StatusTooManyRequests, util.ResponseError("rate limit exceeded, Please try again after 1 minutes"))
			return
		}

		c.Next()
	}
}
	