package handler

import (
	"url-shortener/internal/middleware"
	"url-shortener/internal/service"

	"github.com/gin-gonic/gin"
)

func RegisterRoutes(r *gin.RouterGroup) {
	u := r.Group("/user")
	u.POST("/register", service.RegisterUser)
	u.POST("/login", service.LoginUser)
	u.GET("/get-user", middleware.AuthRequired(), service.GetCurrentUser)
}
