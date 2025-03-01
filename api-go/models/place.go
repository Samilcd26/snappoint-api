package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

type Place struct {
	gorm.Model
	ID          uint           `json:"id" gorm:"primaryKey;autoIncrement"`
	Name        string         `json:"name" gorm:"not null"`
	Description string         `json:"description" gorm:"type:text"`
	Categories  pq.StringArray `json:"categories" gorm:"type:text[]"`
	Address     string         `json:"address" gorm:"not null"`
	Latitude    float64        `json:"latitude" gorm:"not null;type:decimal(10,8)"`
	Longitude   float64        `json:"longitude" gorm:"not null;type:decimal(11,8)"`
	Rating      float64        `json:"rating" gorm:"not null;default:0;type:decimal(3,2)"`
	PointValue  int            `json:"pointValue" gorm:"not null;default:0"`
	PlaceType   string         `json:"placeType" gorm:"not null"`
	PlaceImage  string         `json:"placeImage"`
	OpeningTime *string        `json:"openingTime" gorm:"type:varchar(255);null"`
	ClosingTime *string        `json:"closingTime" gorm:"type:varchar(255);null"`
	IsVerified  bool           `json:"isVerified" gorm:"default:false"`
	Website     string         `json:"website"`
	Phone       string         `json:"phone"`
	PriceLevel  int            `json:"priceLevel" gorm:"type:int;check:price_level between 1 and 4"`
	Features    pq.StringArray `json:"features" gorm:"type:text[]"` // ["wifi", "parking", "outdoor_seating"]
	Posts       []Post         `json:"posts" gorm:"foreignKey:PlaceID"`
	TotalVisits int            `json:"totalVisits" gorm:"default:0"`
	CreatedAt   time.Time      `json:"createdAt"`
	UpdatedAt   time.Time      `json:"updatedAt"`
	IsGenerated bool           `json:"isGenerated" gorm:"default:false"`
	Distance    float64        `json:"distance" gorm:"-"`
}
