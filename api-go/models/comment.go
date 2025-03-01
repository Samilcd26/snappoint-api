package models

import (
	"time"

	"gorm.io/gorm"
)

type Comment struct {
	gorm.Model
	Content   string
	UserID    uint
	User      User
	PostID    uint
	Post      Post
	CreatedAt time.Time
	UpdatedAt time.Time
}
