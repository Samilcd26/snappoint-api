package models

import (

    "gorm.io/gorm"
)

type Follow struct {
    gorm.Model
    FollowerUserID  uint   `gorm:"not null"`
    FollowingUserID uint   `gorm:"not null"`
    Status          string `gorm:"not null;default:'pending'"` // pending, accepted, blocked

    FollowerUser  User `gorm:"foreignKey:FollowerUserID"`
    FollowingUser User `gorm:"foreignKey:FollowingUserID"`
}