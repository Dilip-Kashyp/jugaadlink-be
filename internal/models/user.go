package models

import "gorm.io/gorm"

type User struct {
	gorm.Model
	Email         string  `json:"email" gorm:"unique" validate:"required,email"`
	Name          string  `json:"name" validate:"required"`
	Password      string  `json:"password,omitempty" validate:"omitempty,min=6"`
	OAuthProvider string  `json:"oauth_provider,omitempty" gorm:"default:null"`
	OAuthID       string  `json:"oauth_id,omitempty" gorm:"default:null"`
	AvatarURL     string  `json:"avatar_url,omitempty" gorm:"default:null"`
}
