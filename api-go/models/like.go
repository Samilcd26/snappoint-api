package models

import (
    "time"
)

type Like struct {
    LikeID    uint      `gorm:"column:like_id;primaryKey;autoIncrement"`
    PostID    uint      `gorm:"column:post_id;not null"`
    UserID    uint      `gorm:"column:user_id;not null"`
    CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`

    // İlişkiler
    User User `gorm:"foreignKey:UserID"`
    Post Post `gorm:"foreignKey:PostID"`
}