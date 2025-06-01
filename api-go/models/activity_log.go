package models

import (
	"time"

	"gorm.io/gorm"
)

type ActivityLog struct {
	gorm.Model
	CreatedAt time.Time `json:"createdAt"`
	UserID    uint      `json:"userId" gorm:"not null"`
	User      User      `json:"user" gorm:"foreignKey:UserID"`
	PlaceID   uint      `json:"placeId" gorm:"not null"`
	Place     Place     `json:"place" gorm:"foreignKey:PlaceID"`
	PostID    uint      `json:"postId"`
	Post      Post      `json:"post" gorm:"foreignKey:PostID"`
	Activity  string    `json:"activity" gorm:"not null;type:varchar(50)"` // "post_created", "place_visited", etc.
	Points    int       `json:"points" gorm:"not null;default:0"`
	Latitude  float64   `json:"latitude" gorm:"not null;type:decimal(10,8)"`
	Longitude float64   `json:"longitude" gorm:"not null;type:decimal(11,8)"`
}
