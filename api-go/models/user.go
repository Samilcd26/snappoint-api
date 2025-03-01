package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Username      string `gorm:"unique;not null"`
	FirstName     string
	LastName      string
	Email         string `gorm:"unique;not null"`
	Phone         string `gorm:"unique;not null"`
	Password      string `gorm:"not null"`
	Bio           string
	Avatar        string
	Posts         []Post
	Comments      []Comment
	Likes         []Like
	Followers     []Follow `gorm:"foreignKey:FollowingID"`
	Following     []Follow `gorm:"foreignKey:FollowerID"`
	Roles         []Role   `gorm:"many2many:user_roles;"`
	RefreshTokens []RefreshToken
	CreatedAt     time.Time
	UpdatedAt     time.Time
}
