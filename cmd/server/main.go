package main

import (
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
		AllowOrigins:     []string{"http://localhost:3000", "http://127.0.0.1:3000", "http://localhost:8080"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "OPTIONS"},
		AllowHeaders:     []string{"Content-Type", "Authorization", "X-Session-Token"},
		ExposeHeaders:    []string{"X-Session-Token"},
		AllowCredentials: true,
	}))
	r.Use(middleware.Logger())

	config.ConnectDB()
	config.ConnectRedis()
	config.DB.AutoMigrate(&models.User{}, &models.URL{}, &models.GuestSession{}, &models.Click{})

	v1 := r.Group("/api/v1")
	public := r.Group("/")
	handler.PingRoutes(v1)
	handler.RegisterRoutes(v1)
	handler.URLRoutes(v1)
	handler.RedirectURLRoutes(public)

	// go func() {
	// 	ticker := time.NewTicker(24 * time.Hour) // Run daily
	// 	defer ticker.Stop()
	// 	for {
	// 		select {
	// 		case <-ticker.C:
	// 			utils.CleanupExpiredData()
	// 		}
	// 	}
	// }()

	r.Run(":8080")
}
