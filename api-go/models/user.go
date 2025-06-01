package models

import (
	"time"

	"gorm.io/gorm"
)

type User struct {
	ID            uint           `gorm:"primaryKey;autoIncrement" json:"id"`
	CreatedAt     time.Time      `json:"created_at"`
	UpdatedAt     time.Time      `json:"updated_at"`
	DeletedAt     gorm.DeletedAt `gorm:"index" json:"deleted_at"`
	Username      string         `gorm:"unique;not null" json:"username"`
	FirstName     string         `json:"first_name"`
	LastName      string         `json:"last_name"`
	Gender        string         `json:"gender"`
	Birthday      time.Time      `json:"birthday"`
	Email         string         `gorm:"unique;not null" json:"email"`
	Phone         string         `gorm:"unique;not null" json:"phone"`
	Password      string         `gorm:"not null" json:"-"` // Don't expose password in JSON
	Bio           string         `json:"bio"`
	Avatar        string         `json:"avatar"`
	Posts         []Post         `json:"posts" gorm:"foreignKey:UserID"`
	Comments      []Comment      `json:"comments" gorm:"foreignKey:UserID"`
	Likes         []Like         `json:"likes" gorm:"foreignKey:UserID"`
	Followers     []User         `json:"followers" gorm:"many2many:follows;foreignKey:ID;joinForeignKey:FollowingUserID;References:ID;joinReferences:FollowerUserID"`
	Following     []User         `json:"following" gorm:"many2many:follows;foreignKey:ID;joinForeignKey:FollowerUserID;References:ID;joinReferences:FollowingUserID"`
	Role          Role           `json:"role" gorm:"foreignKey:RoleID"`
	RoleID        uint           `json:"role_id"`
	RefreshTokens []RefreshToken `json:"refresh_tokens" gorm:"foreignKey:UserID"`
	AccountStatus string         `json:"account_status"`
	IsVerified    bool           `json:"is_verified"`
	EmailVerified bool           `json:"email_verified"`
	PhoneVerified bool           `json:"phone_verified"`
	TotalPoints   int64          `gorm:"default:0" json:"total_points"`
}
