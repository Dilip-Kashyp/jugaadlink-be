package handler

import (
	"url-shortener/internal/middleware"
	"url-shortener/internal/service"

	"github.com/gin-gonic/gin"
)

func URLRoutes(r *gin.RouterGroup) {
	u := r.Group("/url")
	u.POST("/shorten", middleware.ResolveIdentity(), middleware.RateLimiter(), service.ShortenURL)
	u.GET("/history", middleware.ResolveIdentity(), service.GetHistory)
	u.DELETE("/:code", middleware.AuthRequired(), service.DeleteURL)
	u.GET("/analytics", middleware.AuthRequired(), service.GetDashboardAnalytics)
	u.GET("/analytics/:code", middleware.AuthRequired(), service.GetURLStats)
	u.GET("/preview", middleware.ResolveIdentity(), service.GetLinkPreview)
}

func RedirectURLRoutes(r *gin.RouterGroup) {
	r.GET("/:code", service.RedirectURL)
}
