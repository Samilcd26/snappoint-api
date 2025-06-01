package models

import (
    "time"

)

type Comment struct {
    CommentID       uint      `gorm:"column:comment_id;primaryKey;autoIncrement"`
    PostID          uint      `gorm:"column:post_id;not null"`
    UserID          uint      `gorm:"column:user_id;not null"`
    ParentCommentID *uint     `gorm:"column:parent_comment_id"` // yanıtlar için isteğe bağlı üst yorum
    TextContent     string    `gorm:"column:text_content;type:text;not null"`
    CreatedAt       time.Time `gorm:"column:created_at;autoCreateTime"`
    IsEdited        bool      `gorm:"column:is_edited;default:false"`
    LikeCount       int       `gorm:"column:like_count;default:0"`

    // İlişkiler
    ParentComment *Comment `gorm:"foreignKey:ParentCommentID"`
    User          User     `gorm:"foreignKey:UserID"`
    Post          Post     `gorm:"foreignKey:PostID"`
}