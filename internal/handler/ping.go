package handler

import (
	"url-shortener/internal/middleware"
	"url-shortener/internal/service"

	"github.com/gin-gonic/gin"
)

func PingRoutes(r *gin.RouterGroup) {
	u := r.Group("/test")
	u.GET("/ping", middleware.RateLimiter(), service.PingService)
	u.POST("/seed", service.SeedData)
	u.POST("/seed/reset", service.ResetSeedData)
}
