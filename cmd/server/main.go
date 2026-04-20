package main

import (
	"os"
	"strings"
	"url-shortener/internal/config"
	"url-shortener/internal/handler"
	"url-shortener/internal/middleware"
	"url-shortener/internal/models"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
)

func main() {
	gin.SetMode(gin.ReleaseMode)
	r := gin.Default()

	r.Use(cors.New(cors.Config{
		AllowOriginFunc: func(origin string) bool {
			if strings.HasPrefix(origin, "chrome-extension://") {
				return true
			}
			return origin == "https://jugaadlink.vercel.app"
		},
		AllowMethods: []string{"GET", "POST", "PUT", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders: []string{"Origin", "Content-Type", "Authorization", "X-Session-Token"},
		ExposeHeaders: []string{"X-Session-Token"},
		AllowCredentials: true,
	}))
	r.Use(middleware.Logger())

	config.ConnectDB()
	config.ConnectRedis()
	config.DB.AutoMigrate(&models.User{}, &models.URL{}, &models.GuestSession{}, &models.Click{})

	v1 := r.Group("/api/v1")
	public := r.Group("/")
	auth := r.Group("/auth") // matches /auth/google/callback and /auth/github/callback
	handler.PingRoutes(v1)
	handler.RegisterRoutes(v1)
	handler.URLRoutes(v1)
	handler.RedirectURLRoutes(public)
	handler.OAuthRoutes(auth)

	// Determine port
	port := os.Getenv("APP_PORT")
	if port == "" {
		port = "8080"
	}
	r.Run(":" + port)
}

