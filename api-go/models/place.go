package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Place struct {
	ID                uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt         time.Time      `json:"created_at"`
	UpdatedAt         time.Time     `json:"updated_at"`
	DeletedAt         gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Name              string         `json:"name" gorm:"not null"`
	Categories        pq.StringArray `json:"categories" gorm:"type:text[]"`
	Address           string         `json:"address" gorm:"not null"`
	Latitude          float64        `json:"latitude" gorm:"not null;type:decimal(10,8)"`
	Longitude         float64        `json:"longitude" gorm:"not null;type:decimal(11,8)"`
	BasePoints        int            `json:"base_points" gorm:"not null;default:0"`
	PlaceType         string         `json:"place_type" gorm:"not null"`
	PlaceImage        string         `json:"place_image" gorm:"type:text"`
	IsVerified        bool           `json:"is_verified" gorm:"default:false"`
	Features          pq.StringArray `json:"features" gorm:"type:text[]"`
	GooglePlaceID     string         `json:"google_place_id" gorm:"type:varchar(255);uniqueIndex"`
	Rating            *float64       `json:"rating" gorm:"type:decimal(2,1)"`
	UserRatingsTotal  *int           `json:"user_ratings_total"`
	BusinessStatus    string         `json:"business_status" gorm:"type:varchar(50)"`
	Icon              string         `json:"icon" gorm:"type:text"`
	PhotoReferences   pq.StringArray `json:"photo_references" gorm:"type:text[]"`
	PlusCode          string         `json:"plus_code" gorm:"type:varchar(20)"`
	Phone             string         `json:"phone" gorm:"type:varchar(20)"`
	Website           string         `json:"website" gorm:"type:text"`
	PriceLevel        *int           `json:"price_level" gorm:"type:smallint"`
	OpeningHours      *string        `json:"opening_hours" gorm:"type:jsonb"`
	Posts             []Post         `json:"posts" gorm:"foreignKey:PlaceID"`
}
