package util

import (
	"crypto/rand"
	"encoding/base64"
	"strconv"
	"url-shortener/internal/config"
	"url-shortener/internal/models"

	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"
)

func GenerateShortCode() string {
	bytes := make([]byte, 6)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)[:8]
}

func ParseInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func ResponseSuccess(data interface{}) models.APIResponse {
	return models.APIResponse{
		Status: true,
		Data:   data,
	}
}

func ResponseError(message string) models.APIResponse {
	return models.APIResponse{
		Status:  false,
		Message: message,
	}
}

func CleanupExpiredData() {
	config.DB.Where("expires_at IS NOT NULL AND expires_at < ?", time.Now()).Delete(&models.URL{})

	config.DB.Where("expires_at < ?", time.Now()).Delete(&models.GuestSession{})

	oneYearAgo := time.Now().AddDate(-1, 0, 0)
	config.DB.Where("created_at < ?", oneYearAgo).Delete(&models.Click{})
}

func GetTokenClaims(c *gin.Context) (jwt.MapClaims, bool) {
	claims, exists := c.Get("token_claims")
	if !exists {
		return nil, false
	}

	tokenClaims, ok := claims.(jwt.MapClaims)
	return tokenClaims, ok
}

func GetUserID(c *gin.Context) (uint, bool) {
	// ResolveIdentity middleware sets "user_id" directly on the context.
	if uid, exists := c.Get("user_id"); exists {
		if id, ok := uid.(uint); ok && id != 0 {
			return id, true
		}
	}

	// AuthRequired middleware sets "token_claims" on the context.
	claims, ok := GetTokenClaims(c)
	if !ok {
		return 0, false
	}

	id, ok := claims["user_id"].(float64)
	if !ok {
		return 0, false
	}

	return uint(id), true
}

func GetUserEmail(c *gin.Context) (string, bool) {
	claims, ok := GetTokenClaims(c)
	if !ok {
		return "", false
	}

	email, ok := claims["email"].(string)
	return email, ok
}
