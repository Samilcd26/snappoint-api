package models

import (
	"time"
	"gorm.io/gorm"
)

type Report struct {
	ID             uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt      time.Time      `json:"created_at"`
	UpdatedAt      time.Time      `json:"updated_at"`
	DeletedAt      gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	
	ReporterUserID uint   `gorm:"not null" json:"reporter_user_id"`
	ReportedUserID uint   `gorm:"not null" json:"reported_user_id"`
	Reason         string `gorm:"not null" json:"reason"`
	Description    string `json:"description"`
	Status         string `gorm:"not null;default:'pending'" json:"status"` // pending, reviewed, resolved, dismissed
	
	ReporterUser User `gorm:"foreignKey:ReporterUserID" json:"reporter_user"`
	ReportedUser User `gorm:"foreignKey:ReportedUserID" json:"reported_user"`
} 