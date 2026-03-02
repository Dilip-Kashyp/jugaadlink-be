package handler

import (
	"url-shortener/internal/middleware"
	"url-shortener/internal/service"

	"github.com/gin-gonic/gin"
)

func URLRoutes(r *gin.RouterGroup) {
	u := r.Group("/url")
	u.POST("/shorten", middleware.ResolveIdentity(), middleware.RateLimiter(), service.ShortenURL)
	u.GET("/history", middleware.AuthRequired(), service.GetHistory)
	u.GET("/redirect/:code", service.RedirectURL) // public route
	u.DELETE("/:code", middleware.AuthRequired(), service.DeleteURL)
	u.GET("/stats/:code", middleware.ResolveIdentity(), service.GetURLStats)
}
