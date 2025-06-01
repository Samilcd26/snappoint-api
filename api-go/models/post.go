package models

import (
	"time"

	"gorm.io/gorm"
)

type Post struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	PostCaption   string         `json:"post_caption" gorm:"type:text"`
	UserID        uint           `json:"user_id" gorm:"not null"`
	PlaceID       uint           `json:"place_id" gorm:"not null"`
	EarnedPoints  int64          `json:"earned_points" gorm:"not null;default:0"`
	User          User           `json:"user" gorm:"foreignKey:UserID"`
	Place         Place          `json:"place" gorm:"foreignKey:PlaceID"`
	Latitude      float64        `json:"latitude" gorm:"type:decimal(10,8)"`
	Longitude     float64        `json:"longitude" gorm:"type:decimal(11,8)"`
	IsArchived    bool           `json:"is_archived" gorm:"default:false"`
	AllowComments bool           `json:"allow_comments" gorm:"default:true"`
	IsPublic      bool           `json:"is_public" gorm:"default:true"`
	PostMedia     []PostMedia    `json:"post_media" gorm:"foreignKey:PostID"`
	Comments      []Comment      `json:"comments" gorm:"foreignKey:PostID"`
	Likes         []Like         `json:"likes" gorm:"foreignKey:PostID"`
}
