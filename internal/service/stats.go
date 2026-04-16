package service

import (
	"net/http"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
)

func GetURLStats(c *gin.Context) {
	shortCode := c.Param("code")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, util.ResponseError("code is required"))
		return
	}

	var url models.URL
	if err := config.DB.Where("short_code = ?", shortCode).First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	// Basic authorization check
	if userID, ok := c.Get("user_id"); ok {
		userIDValue := userID.(uint)
		if url.UserID == nil || *url.UserID != userIDValue {
			c.JSON(http.StatusForbidden, util.ResponseError("access denied"))
			return
		}
	} else {
		sessionID := c.GetUint("session_id")
		if url.SessionID == nil || *url.SessionID != sessionID {
			c.JSON(http.StatusForbidden, util.ResponseError("access denied"))
			return
		}
	}

	var stats struct {
		TotalClicks int64                  `json:"total_clicks"`
		Browsers    []map[string]interface{} `json:"browsers"`
		Devices     []map[string]interface{} `json:"devices"`
		OS         []map[string]interface{} `json:"os"`
		Regions   []map[string]interface{} `json:"regions"`
		Referrers   []map[string]interface{} `json:"referrers"`
		Timeline    []map[string]interface{} `json:"timeline"`
	}

	stats.TotalClicks = int64(url.Clicks)

	// Browser breakdown
	config.DB.Model(&models.Click{}).
		Select("browser as name, count(*) as value").
		Where("url_id = ?", url.ID).
		Group("browser").
		Scan(&stats.Browsers)

	// Device breakdown
	config.DB.Model(&models.Click{}).
		Select("device as name, count(*) as value").
		Where("url_id = ?", url.ID).
		Group("device").
		Scan(&stats.Devices)

	// OS breakdown
	config.DB.Model(&models.Click{}).
		Select("os as name, count(*) as value").
		Where("url_id = ?", url.ID).
		Group("os").
		Scan(&stats.OS)

	// Region breakdown
	config.DB.Model(&models.Click{}).
		Select("country as name, count(*) as value").
		Where("url_id = ?", url.ID).
		Group("country").
		Scan(&stats.Regions)

	// Referrer breakdown
	config.DB.Model(&models.Click{}).
		Select("referer as name, count(*) as value").
		Where("url_id = ?", url.ID).
		Group("referer").
		Scan(&stats.Referrers)

	// Timeline (last 7 days)
	config.DB.Raw(`
		SELECT TO_CHAR(timestamp, 'YYYY-MM-DD') as name, COUNT(*) as clicks
		FROM clicks
		WHERE url_id = ? AND timestamp > NOW() - INTERVAL '7 days'
		GROUP BY name
		ORDER BY name ASC
	`, url.ID).Scan(&stats.Timeline)

	c.JSON(http.StatusOK, util.ResponseSuccess(stats))
}
