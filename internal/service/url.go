package service

import (
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strings"
	"time"
	"url-shortener/internal/config"
	"url-shortener/internal/models"
	"url-shortener/internal/util"

	"github.com/gin-gonic/gin"
	"github.com/mssola/useragent"
	"golang.org/x/crypto/bcrypt"
	"golang.org/x/net/html"
	"gorm.io/gorm"
)

func ShortenURL(c *gin.Context) {
	var input struct {
		OriginalURL  string     `json:"original_url" validate:"required,url"`
		CustomSlug   string     `json:"custom_slug"`
		ExpiresAt    *time.Time `json:"expires_at"`
		Password     string     `json:"password"`
		MaxClicks    int        `json:"max_clicks"`
		Tags         string     `json:"tags"`
		Category     string     `json:"category"`
		Comment      string     `json:"comment"`
		CustomDomain string     `json:"custom_domain"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, util.ResponseError(err.Error()))
		return
	}

	shortCode := ""
	if input.CustomSlug != "" {
		// Basic validation for custom slug
		if len(input.CustomSlug) < 3 || len(input.CustomSlug) > 20 {
			c.JSON(http.StatusBadRequest, util.ResponseError("custom slug must be between 3 and 20 characters"))
			return
		}
		// Check if it already exists
		var existing models.URL
		if err := config.DB.Where("short_code = ?", input.CustomSlug).First(&existing).Error; err == nil {
			c.JSON(http.StatusConflict, util.ResponseError("custom slug already in use"))
			return
		}
		shortCode = input.CustomSlug
	} else {
		shortCode = util.GenerateShortCode()
	}

	url := models.URL{
		OriginalURL:  input.OriginalURL,
		ShortCode:    shortCode,
		ExpiresAt:    input.ExpiresAt,
		MaxClicks:    input.MaxClicks,
		Tags:         input.Tags,
		Category:     input.Category,
		Comment:      input.Comment,
		CustomDomain: input.CustomDomain,
	}

	if input.Password != "" {
		hashed, _ := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
		url.Password = string(hashed)
	}

	// Fetch OG Tags
	title, desc, img := fetchOGTags(input.OriginalURL)
	url.Title = title
	url.Description = desc
	url.Image = img

	if userID, ok := c.Get("user_id"); ok {
		userIDValue := userID.(uint)
		// Validate that the user exists before associating the URL
		var user models.User
		if err := config.DB.First(&user, userIDValue).Error; err != nil {
			c.JSON(http.StatusUnauthorized, util.ResponseError("user not found"))
			return
		}
		url.UserID = &userIDValue
	} else {
		sessionID := c.GetUint("session_id")
		url.SessionID = &sessionID
	}

	if err := config.DB.Create(&url).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.ResponseError(err.Error()))
		return
	}

	c.JSON(http.StatusCreated, util.ResponseSuccess(gin.H{
		"short_code":    shortCode,
		"short_url":     os.Getenv("SERVER_URL") + shortCode,
		"original_url":  input.OriginalURL,
		"tags":          input.Tags,
		"category":      input.Category,
		"comment":       input.Comment,
		"custom_domain": input.CustomDomain,
		"title":         url.Title,
	}))
}

func RedirectURL(c *gin.Context) {
	shortCode := c.Param("code")
	frontendURL := strings.TrimRight(os.Getenv("FRONTEND_URL"), "/")
	if frontendURL == "" {
		frontendURL = "http://localhost:3000"
	}

	var url models.URL
	if err := config.DB.Where("short_code = ?", shortCode).First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	// 0. Check if link is active
	if !url.IsActive {
		c.Redirect(http.StatusFound, frontendURL+"/link-disabled?code="+shortCode)
		return
	}

	// 1. Expiry check (time-based)
	if url.ExpiresAt != nil && time.Now().After(*url.ExpiresAt) {
		c.Redirect(http.StatusFound, frontendURL+"/link-disabled?code="+shortCode+"&reason=expired")
		return
	}

	// 2. Expiry check (usage-based)
	if url.MaxClicks > 0 && url.Clicks >= url.MaxClicks {
		c.Redirect(http.StatusFound, frontendURL+"/link-disabled?code="+shortCode+"&reason=limit")
		return
	}

	// 3. Password protection — redirect to frontend password page
	if url.Password != "" {
		password := c.GetHeader("X-URL-Password")
		if password == "" {
			password = c.Query("password")
		}

		if password == "" {
			c.Redirect(http.StatusFound, frontendURL+"/password/"+shortCode)
			return
		}

		if err := bcrypt.CompareHashAndPassword([]byte(url.Password), []byte(password)); err != nil {
			c.Redirect(http.StatusFound, frontendURL+"/password/"+shortCode+"?error=invalid")
			return
		}
	}

	// 4. Record Click
	go recordClick(url.ID, c.ClientIP(), c.GetHeader("User-Agent"), c.Request.Referer())

	// 5. Cache
	if url.Password == "" && url.MaxClicks == 0 && url.ExpiresAt == nil && config.RedisClient != nil {
		config.RedisClient.Set(config.RedisClient.Context(), shortCode, url.OriginalURL, time.Hour)
	}

	redirectTo := url.OriginalURL
	if !strings.HasPrefix(redirectTo, "http://") && !strings.HasPrefix(redirectTo, "https://") {
		redirectTo = "https://" + redirectTo
	}

	c.Redirect(http.StatusFound, redirectTo)
}

// VerifyPassword verifies a password for a protected link and returns the redirect URL
func VerifyPassword(c *gin.Context) {
	shortCode := c.Param("code")
	var input struct {
		Password string `json:"password"`
	}
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, util.ResponseError("password is required"))
		return
	}

	var url models.URL
	if err := config.DB.Where("short_code = ?", shortCode).First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(url.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, util.ResponseError("incorrect password"))
		return
	}

	// Record click
	go recordClick(url.ID, c.ClientIP(), c.GetHeader("User-Agent"), c.Request.Referer())

	redirectTo := url.OriginalURL
	if !strings.HasPrefix(redirectTo, "http://") && !strings.HasPrefix(redirectTo, "https://") {
		redirectTo = "https://" + redirectTo
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"redirect_url": redirectTo,
	}))
}

// UpdateURL allows the owner of a link to update its metadata after creation
func UpdateURL(c *gin.Context) {
	shortCode := c.Param("code")

	var input struct {
		Password  *string    `json:"password"`
		ExpiresAt *time.Time `json:"expires_at"`
		MaxClicks *int       `json:"max_clicks"`
		Tags      *string    `json:"tags"`
		Category  *string    `json:"category"`
		Comment   *string    `json:"comment"`
		IsActive  *bool      `json:"is_active"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, util.ResponseError(err.Error()))
		return
	}

	query := config.DB.Where("short_code = ?", shortCode)

	if userID, ok := util.GetUserID(c); ok {
		query = query.Where("user_id = ?", userID)
	} else {
		sessionID := c.GetUint("session_id")
		if sessionID == 0 {
			c.JSON(http.StatusUnauthorized, util.ResponseError("unauthorized"))
			return
		}
		query = query.Where("session_id = ?", sessionID)
	}

	var url models.URL
	if err := query.First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	updates := map[string]interface{}{}

	if input.Password != nil {
		if *input.Password == "" {
			// Clear the password
			updates["password"] = ""
		} else {
			hashed, err := bcrypt.GenerateFromPassword([]byte(*input.Password), bcrypt.DefaultCost)
			if err != nil {
				c.JSON(http.StatusInternalServerError, util.ResponseError("failed to hash password"))
				return
			}
			updates["password"] = string(hashed)
		}
	}
	if input.ExpiresAt != nil {
		updates["expires_at"] = input.ExpiresAt
	}
	if input.MaxClicks != nil {
		updates["max_clicks"] = *input.MaxClicks
	}
	if input.Tags != nil {
		updates["tags"] = *input.Tags
	}
	if input.Category != nil {
		updates["category"] = *input.Category
	}
	if input.Comment != nil {
		updates["comment"] = *input.Comment
	}
	if input.IsActive != nil {
		updates["is_active"] = *input.IsActive
	}

	if len(updates) == 0 {
		c.JSON(http.StatusBadRequest, util.ResponseError("no fields to update"))
		return
	}

	if err := config.DB.Model(&url).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.ResponseError(err.Error()))
		return
	}

	// Invalidate Redis cache on update so redirects pick up the new state
	if config.RedisClient != nil {
		config.RedisClient.Del(c.Request.Context(), shortCode)
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"message":    "Link updated successfully",
		"short_code": shortCode,
	}))
}


func ToggleURL(c *gin.Context) {
	shortCode := c.Param("code")

	var url models.URL
	query := config.DB.Where("short_code = ?", shortCode)

	if userID, ok := util.GetUserID(c); ok {
		query = query.Where("user_id = ?", userID)
	} else {
		sessionID := c.GetUint("session_id")
		if sessionID == 0 {
			c.JSON(http.StatusUnauthorized, util.ResponseError("unauthorized"))
			return
		}
		query = query.Where("session_id = ?", sessionID)
	}

	if err := query.First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	newStatus := !url.IsActive
	config.DB.Model(&url).Update("is_active", newStatus)

	// Clear cache if deactivating
	if !newStatus && config.RedisClient != nil {
		config.RedisClient.Del(c.Request.Context(), shortCode)
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"is_active": newStatus,
		"message":   "Link status updated",
	}))
}

func GetHistory(c *gin.Context) {
	var urls []models.URL

	page := util.ParseInt(c.DefaultQuery("page", "1"))
	limit := util.ParseInt(c.DefaultQuery("limit", "10"))

	if page < 1 {
		page = 1
	}
	if limit < 1 {
		limit = 10
	}

	offset := (page - 1) * limit

	query := config.DB.Model(&models.URL{})

	if userID, ok := c.Get("user_id"); ok {
		query = query.Where("user_id = ?", userID)
	} else {
		sessionID := c.GetUint("session_id")
		if sessionID == 0 {
			c.JSON(http.StatusUnauthorized, util.ResponseError("unauthorized"))
			return
		}
		query = query.Where("session_id = ?", sessionID)
	}

	if err := query.
		Order("created_at DESC").
		Limit(limit).
		Offset(offset).
		Find(&urls).Error; err != nil {

		c.JSON(http.StatusInternalServerError, util.ResponseError(err.Error()))
		return
	}

	history := make([]models.HistoryItem, 0, len(urls))
	for _, u := range urls {
		history = append(history, models.HistoryItem{
			ID:           u.ID,
			OriginalURL:  u.OriginalURL,
			ShortCode:    u.ShortCode,
			ShortURL:     os.Getenv("SERVER_URL") + u.ShortCode,
			Clicks:       u.Clicks,
			MaxClicks:    u.MaxClicks,
			HasPassword:  u.Password != "",
			ExpiresAt:    u.ExpiresAt,
			CreatedAt:    u.CreatedAt,
			Tags:         u.Tags,
			Category:     u.Category,
			Comment:      u.Comment,
			CustomDomain: u.CustomDomain,
			Title:        u.Title,
			Description:  u.Description,
			Image:        u.Image,
			IsActive:     u.IsActive,
		})
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"history": history,
		"meta": gin.H{
			"page":  page,
			"limit": limit,
			"count": len(history),
		},
	}))
}

func DeleteURL(c *gin.Context) {
	shortCode := c.Param("code")
	if shortCode == "" {
		c.JSON(http.StatusBadRequest, util.ResponseError("code is required"))
		return
	}

	var url models.URL
	query := config.DB.Where("short_code = ?", shortCode)

	// NEW AUTH FLOW
	if userID, ok := util.GetUserID(c); ok {
		query = query.Where("user_id = ?", userID)
	} else {
		sessionID := c.GetUint("session_id")
		if sessionID == 0 {
			c.JSON(http.StatusUnauthorized, util.ResponseError("unauthorized"))
			return
		}
		query = query.Where("session_id = ?", sessionID)
	}

	if err := query.First(&url).Error; err != nil {
		c.JSON(http.StatusNotFound, util.ResponseError("URL not found"))
		return
	}

	if err := config.DB.Where("url_id = ?", url.ID).
		Delete(&models.Click{}).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.ResponseError(err.Error()))
		return
	}

	if err := config.DB.Delete(&url).Error; err != nil {
		c.JSON(http.StatusInternalServerError, util.ResponseError(err.Error()))
		return
	}

	// Clear cache (skip if Redis is not available)
	if config.RedisClient != nil {
		config.RedisClient.Del(c.Request.Context(), shortCode)
	}

	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"message": "URL deleted successfully",
	}))
}

// isPrivateIP reports whether ip is a loopback or private-range address
// that ip-api.com cannot resolve (Docker bridge, localhost, etc.).
func isPrivateIP(ip string) bool {
	private := []string{"127.", "::1", "10.", "192.168.", "172.16.", "172.17.", "172.18.", "172.19.", "172.20.", "172.21.", "172.22.", "172.23.", "172.24.", "172.25.", "172.26.", "172.27.", "172.28.", "172.29.", "172.30.", "172.31."}
	for _, prefix := range private {
		if strings.HasPrefix(ip, prefix) {
			return true
		}
	}
	return false
}

func recordClick(urlID uint, ip, ua, referer string) {
	uaParser := useragent.New(ua)
	browserName, browserVersion := uaParser.Browser()
	click := models.Click{
		URLID:     urlID,
		IP:        ip,
		UserAgent: ua,
		Browser:   fmt.Sprintf("%s %s", browserName, browserVersion),
		OS:        uaParser.OS(),
		Device:    "Desktop",
		Referer:   referer,
	}

	if uaParser.Mobile() {
		click.Device = "Mobile"
	} else if uaParser.Bot() {
		click.Device = "Bot"
	}

	// Geo-IP lookup — skip for private/local addresses (Docker bridge, localhost)
	if !isPrivateIP(ip) {
		resp, err := http.Get(fmt.Sprintf("http://ip-api.com/json/%s", ip))
		if err == nil {
			defer resp.Body.Close()
			var geo struct {
				Country string `json:"country"`
				City    string `json:"city"`
			}
			if err := json.NewDecoder(resp.Body).Decode(&geo); err == nil {
				click.Country = geo.Country
				click.City = geo.City
			}
		}
	}

	if err := config.DB.Create(&click).Error; err != nil {
		// log error
	}

	config.DB.Model(&models.URL{}).Where("id = ?", urlID).Update("clicks", gorm.Expr("clicks + 1"))
}

func fetchOGTags(urlStr string) (string, string, string) {
	// Ensure protocol scheme
	if !strings.HasPrefix(urlStr, "http://") && !strings.HasPrefix(urlStr, "https://") {
		urlStr = "https://" + urlStr
	}

	client := &http.Client{
		Timeout: 10 * time.Second,
	}

	req, err := http.NewRequest("GET", urlStr, nil)
	if err != nil {
		return "", "", ""
	}

	// Add browser-like headers to avoid being blocked by LinkedIn/scraping protections
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36")
	req.Header.Set("Accept", "text/html,application/xhtml+xml,application/xml;q=0.9,image/avif,image/webp,image/apng,*/*;q=0.8")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")

	resp, err := client.Do(req)
	if err != nil {
		return "", "", ""
	}
	defer resp.Body.Close()

	tokenizer := html.NewTokenizer(resp.Body)
	var title, desc, img string

	for {
		tt := tokenizer.Next()
		if tt == html.ErrorToken {
			break
		}

		if tt == html.StartTagToken || tt == html.SelfClosingTagToken {
			t := tokenizer.Token()
			if t.Data == "meta" {
				var property, content string
				for _, attr := range t.Attr {
					if attr.Key == "property" || attr.Key == "name" {
						property = attr.Val
					}
					if attr.Key == "content" {
						content = attr.Val
					}
				}

				switch property {
				case "og:title", "twitter:title":
					if title == "" {
						title = content
					}
				case "og:description", "description", "twitter:description":
					if desc == "" {
						desc = content
					}
				case "og:image", "twitter:image":
					if img == "" {
						img = content
					}
				}
			} else if t.Data == "title" && title == "" {
				tokenizer.Next()
				title = tokenizer.Token().Data
			}
		}
	}

	return title, desc, img
}

func GetLinkPreview(c *gin.Context) {
	urlStr := c.Query("url")
	if urlStr == "" {
		c.JSON(http.StatusBadRequest, util.ResponseError("URL is required"))
		return
	}

	title, desc, img := fetchOGTags(urlStr)
	c.JSON(http.StatusOK, util.ResponseSuccess(gin.H{
		"title":       title,
		"description": desc,
		"image":       img,
	}))
}
