package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Place struct {
	ID         uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt  time.Time      `json:"created_at"`
	UpdatedAt  time.Time      `json:"updated_at"`
	DeletedAt  gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name       string         `json:"name" gorm:"not null"`
	Categories pq.StringArray `json:"categories" gorm:"type:text[]"`
	Address    string         `json:"address" gorm:"not null"`
	Latitude   float64        `json:"latitude" gorm:"not null;type:decimal(10,8)"`
	Longitude  float64        `json:"longitude" gorm:"not null;type:decimal(11,8)"`
	BasePoints int            `json:"base_points" gorm:"not null;default:0"`
	PlaceType  string         `json:"place_type" gorm:"not null"`
	PlaceImage string         `json:"place_image" gorm:"type:text"`
	IsVerified bool           `json:"is_verified" gorm:"default:false"`
	Features   pq.StringArray `json:"features" gorm:"type:text[]"`
	Posts      []Post         `json:"posts" gorm:"foreignKey:PlaceID"`
}
