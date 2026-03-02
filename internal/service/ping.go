package service

import (
	"net/http"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
)

func PingService(c *gin.Context) {
	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"message": "ping pong",
	}))
}
