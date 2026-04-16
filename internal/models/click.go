package models

import (
	"time"

	"gorm.io/gorm"
)

type Click struct {
	gorm.Model
	URLID     uint      `json:"url_id" gorm:"not null;index"`
	IP        string    `json:"ip"`
	UserAgent string    `json:"user_agent"`
	Device    string    `json:"device"`
	Browser   string    `json:"browser"`
	OS        string    `json:"os"`
	Country   string    `json:"country"`
	City      string    `json:"city"`
	Referer   string    `json:"referer"`
	Timestamp time.Time `json:"timestamp" gorm:"default:CURRENT_TIMESTAMP"`
	URL       URL       `gorm:"foreignKey:URLID"`
}
