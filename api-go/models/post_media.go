package models

import (
	"time"

	"github.com/lib/pq"
	"gorm.io/gorm"
)

// PostMedia represents the media attached to a post.
type PostMedia struct {
	ID           uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt    time.Time      `json:"created_at"`
	UpdatedAt    time.Time      `json:"updated_at"`
	DeletedAt    gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	PostID       uint           `gorm:"not null;index" json:"post_id"`      // Bağlı olduğu gönderi (Foreign Key)
	MediaType    string         `gorm:"size:50;not null" json:"media_type"` // Medya türü (ör: resim, video, ses)
	MediaURL     string         `gorm:"not null" json:"media_url"`          // Medya dosyası linki
	ThumbnailURL string         `json:"thumbnail_url"`                      // Küçük resim (videosa)
	OrderIndex   int            `gorm:"default:0" json:"order_index"`
	Tags         pq.StringArray `json:"tags" gorm:"type:text[]"`
	AltText      string         `gorm:"size:255" json:"alt_text"` // Alternatif metin
	Width        int            `json:"width"`                    // Genişlik
	Height       int            `json:"height"`                   // Yükseklik
	Duration     int            `json:"duration"`                 // Süre (video/ses için, saniye cinsinden)
}
