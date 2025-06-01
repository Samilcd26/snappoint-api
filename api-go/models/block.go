package models

import (
	"time"
	"gorm.io/gorm"
)

type Block struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	
	BlockerUserID uint `gorm:"not null" json:"blocker_user_id"`
	BlockedUserID uint `gorm:"not null" json:"blocked_user_id"`
	
	BlockerUser User `gorm:"foreignKey:BlockerUserID" json:"blocker_user"`
	BlockedUser User `gorm:"foreignKey:BlockedUserID" json:"blocked_user"`
} 