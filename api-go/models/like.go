package models

import (
	"time"

	"gorm.io/gorm"
)

type Like struct {
	gorm.Model
	UserID    uint
	User      User
	PostID    uint
	Post      Post
	CreatedAt time.Time
	UpdatedAt time.Time
}
