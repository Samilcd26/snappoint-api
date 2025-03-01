package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Post struct {
	gorm.Model
	Content    string         `json:"content" gorm:"type:text"`
	MediaType  string         `json:"mediaType" gorm:"not null;type:varchar(10)"` // "photo" or "video"
	MediaURL   pq.StringArray `json:"mediaUrl" gorm:"type:text[]"`
	Hashtags   pq.StringArray `json:"hashtags" gorm:"type:text[]"`
	UserID     uint           `json:"userId" gorm:"not null"`
	User       User           `json:"user" gorm:"foreignKey:UserID"`
	PlaceID    uint           `json:"placeId" gorm:"not null"`
	Place      Place          `json:"place" gorm:"foreignKey:PlaceID"`
	Latitude   float64        `json:"latitude" gorm:"not null;type:decimal(10,8)"`
	Longitude  float64        `json:"longitude" gorm:"not null;type:decimal(11,8)"`
	Points     int            `json:"points" gorm:"not null;default:0"`
	Comments   []Comment      `json:"comments" gorm:"foreignKey:PostID"`
	Likes      []Like         `json:"likes" gorm:"foreignKey:PostID"`
	ViewCount  int            `json:"viewCount" gorm:"default:0"`
	ShareCount int            `json:"shareCount" gorm:"default:0"`
	IsPublic   bool           `json:"isPublic" gorm:"default:true"`
	IsFeatured bool           `json:"isFeatured" gorm:"default:false"`
	Mood       string         `json:"mood" gorm:"type:varchar(50)"`
	Weather    string         `json:"weather" gorm:"type:varchar(50)"`
	Season     string         `json:"season" gorm:"type:varchar(20)"`
	Tags       pq.StringArray `json:"tags" gorm:"type:text[]"` // Additional tags for better categorization
	CreatedAt  time.Time      `json:"createdAt"`
	UpdatedAt  time.Time      `json:"updatedAt"`
}
