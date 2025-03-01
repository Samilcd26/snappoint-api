package models

import (
	"time"

	"gorm.io/gorm"
)

type Follow struct {
	gorm.Model
	FollowerID  uint
	Follower    User
	FollowingID uint
	Following   User
	CreatedAt   time.Time
	UpdatedAt   time.Time
}
