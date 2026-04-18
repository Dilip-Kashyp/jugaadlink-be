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

	// OAuth login initiation — redirects user to provider
	u.GET("/oauth/google", service.GoogleOAuthLogin)
	u.GET("/oauth/github", service.GithubOAuthLogin)
}

// OAuthRoutes registers the provider callback endpoints.
// These MUST match the redirect URIs registered in Google/GitHub consoles.
// Registered as /auth/google/callback and /auth/github/callback.
func OAuthRoutes(r *gin.RouterGroup) {
	r.GET("/google/callback", service.GoogleOAuthCallback)
	r.GET("/github/callback", service.GithubOAuthCallback)
}

