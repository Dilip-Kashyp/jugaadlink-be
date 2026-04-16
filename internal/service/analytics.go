package service

import (
	"fmt"
	"net/http"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
)

func GetDashboardAnalytics(c *gin.Context) {
	interval := c.DefaultQuery("interval", "30d") // 7d, 30d, all

	var timeFilter string
	switch interval {
	case "7d":
		timeFilter = "clicks.timestamp > NOW() - INTERVAL '7 days'"
	case "30d":
		timeFilter = "clicks.timestamp > NOW() - INTERVAL '30 days'"
	case "all":
		timeFilter = "1=1" // No filter
	default:
		timeFilter = "clicks.timestamp > NOW() - INTERVAL '30 days'"
	}

	var userID *uint
	var sessionID *uint

	if uid, ok := c.Get("user_id"); ok {
		val := uid.(uint)
		userID = &val
	} else {
		sid := c.GetUint("session_id")
		if sid == 0 {
			c.JSON(http.StatusUnauthorized, util.ResponseError("unauthorized"))
			return
		}
		sessionID = &sid
	}

	var stats struct {
		TotalUrls   int64                    `json:"total_urls"`
		TotalClicks int64                    `json:"total_clicks"`
		TopUrls     []map[string]interface{} `json:"top_urls"`
		Browsers    []map[string]interface{} `json:"browsers"`
		Devices     []map[string]interface{} `json:"devices"`
		OS          []map[string]interface{} `json:"os"`
		Countries   []map[string]interface{} `json:"countries"`
		Referrers   []map[string]interface{} `json:"referrers"`
		Timeline    []map[string]interface{} `json:"timeline"`
	}

	// Base URL query
	urlQuery := config.DB.Model(&models.URL{})
	if userID != nil {
		urlQuery = urlQuery.Where("user_id = ?", *userID)
	} else if sessionID != nil {
		urlQuery = urlQuery.Where("session_id = ?", *sessionID)
	}

	// Total URLs
	urlQuery.Count(&stats.TotalUrls)

	// Total Clicks
	urlQuery.Select("COALESCE(SUM(clicks), 0)").Scan(&stats.TotalClicks)

	// Top URLs
	urlQuery.
		Select("id, original_url, short_code, title, clicks").
		Order("clicks DESC").
		Limit(5).
		Scan(&stats.TopUrls)

	// Base Click Join Query
	clickJoinQuery := config.DB.Model(&models.Click{}).
		Joins("JOIN urls ON urls.id = clicks.url_id")

	if userID != nil {
		clickJoinQuery = clickJoinQuery.Where("urls.user_id = ?", *userID)
	} else if sessionID != nil {
		clickJoinQuery = clickJoinQuery.Where("urls.session_id = ?", *sessionID)
	}

	// Devices
	clickJoinQuery.
		Select("clicks.device as name, count(*) as value").
		Where(timeFilter).
		Group("clicks.device").
		Scan(&stats.Devices)

	// Browsers
	clickJoinQuery.
		Select("clicks.browser as name, count(*) as value").
		Where(timeFilter).
		Group("clicks.browser").
		Scan(&stats.Browsers)

	// OS
	clickJoinQuery.
		Select("clicks.os as name, count(*) as value").
		Where(timeFilter).
		Group("clicks.os").
		Scan(&stats.OS)

	// Region (Renamed from Countries for FE matching)
	clickJoinQuery.
		Select("clicks.country as name, count(*) as value").
		Where(timeFilter).
		Group("clicks.country").
		Scan(&stats.Countries)

	// Referrers
	clickJoinQuery.
		Select("clicks.referer as name, count(*) as value").
		Where(timeFilter).
		Group("clicks.referer").
		Scan(&stats.Referrers)

	// Timeline
	timelineQuery := fmt.Sprintf(`
		SELECT TO_CHAR(clicks.timestamp, 'YYYY-MM-DD') as name, COUNT(clicks.id) as clicks
		FROM clicks
		JOIN urls ON urls.id = clicks.url_id
		WHERE %s`, timeFilter)

	if userID != nil {
		timelineQuery += fmt.Sprintf(" AND urls.user_id = %d", *userID)
	} else if sessionID != nil {
		timelineQuery += fmt.Sprintf(" AND urls.session_id = %d", *sessionID)
	}

	timelineQuery += " GROUP BY name ORDER BY name ASC"

	config.DB.Raw(timelineQuery).Scan(&stats.Timeline)

	// Manually construct response to match Frontend expected keys
	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"total_urls":   stats.TotalUrls,
		"total_clicks": stats.TotalClicks,
		"top_urls":     stats.TopUrls,
		"browsers":     stats.Browsers,
		"devices":      stats.Devices,
		"os":           stats.OS,
		"regions":      stats.Countries,
		"referrers":    stats.Referrers,
		"timeline":     stats.Timeline,
	}))
}
