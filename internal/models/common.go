package models

import "time"

type APIResponse struct {
	Status  bool        `json:"status"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Meta    interface{} `json:"meta,omitempty"`
}

type HistoryItem struct {
	ID           uint       `json:"id"`
	OriginalURL  string     `json:"original_url"`
	ShortCode    string     `json:"short_code"`
	ShortURL     string     `json:"short_url"`
	Clicks       int        `json:"clicks"`
	MaxClicks    int        `json:"max_clicks"`
	HasPassword  bool       `json:"has_password"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
	CreatedAt    time.Time  `json:"created_at"`
	Tags         string     `json:"tags"`
	Category     string     `json:"category"`
	Comment      string     `json:"comment"`
	CustomDomain string     `json:"custom_domain"`
	Title        string     `json:"title"`
	Description  string     `json:"description"`
	Image        string     `json:"image"`
	IsActive     bool       `json:"is_active"`
}
