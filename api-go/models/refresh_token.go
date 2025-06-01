package models

import (
	"time"

	"gorm.io/gorm"
)

type RefreshToken struct {
	gorm.Model
	CreatedAt      time.Time
	UserID         uint      `json:"userId" gorm:"not null"`
	User           User      `json:"user" gorm:"foreignKey:UserID"`
	Token          string    `json:"token" gorm:"not null"`
	ExpirationDate time.Time `json:"expiry" gorm:"not null"`
}
