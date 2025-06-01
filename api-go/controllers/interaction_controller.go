package controllers

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"gorm.io/gorm"
)

type InteractionController struct {
	DB *gorm.DB
}

func NewInteractionController(db *gorm.DB) *InteractionController {
	return &InteractionController{DB: db}
}

// LikePost godoc
// @Summary Like or unlike a post
// @Description Toggles like status for a post
// @Tags interactions
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} map[string]interface{}
// @Router /posts/{id}/like [post]
func (ic *InteractionController) LikePost(c *gin.Context) {
	postID := c.Param("id")
	userID := c.GetUint("userID") // Assuming this is set by auth middleware

	var post models.Post
	if err := ic.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	var existingLike models.Like
	result := ic.DB.Where("post_id = ? AND user_id = ?", postID, userID).First(&existingLike)

	tx := ic.DB.Begin()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new like
		like := models.Like{
			UserID:    userID,
			PostID:    post.ID,
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&like).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to like post"})
			return
		}

		// Create activity log
		activity := models.ActivityLog{
			UserID:    userID,
			PostID:    post.ID,
			PlaceID:   post.PlaceID,
			Activity:  "post_liked",
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&activity).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity log"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"liked": true})
	} else {
		// Unlike post
		if err := tx.Delete(&existingLike).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unlike post"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{"liked": false})
	}
}

// FollowUser godoc
// @Summary Follow or unfollow a user
// @Description Toggles follow status for a user
// @Tags interactions
// @Accept json
// @Produce json
// @Param userId path string true "User ID to follow"
// @Success 200 {object} map[string]interface{}
// @Router /users/{userId}/follow [post]
func (ic *InteractionController) FollowUser(c *gin.Context) {
	targetUserID := c.Param("userId")
	followerID := c.GetUint("userID") // Assuming this is set by auth middleware

	var targetUser models.User
	if err := ic.DB.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	// Prevent self-following
	if followerID == targetUser.ID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot follow yourself"})
		return
	}

	var existingFollow models.Follow
	result := ic.DB.Where("follower_id = ? AND following_id = ?", followerID, targetUser.ID).First(&existingFollow)

	tx := ic.DB.Begin()

	if result.Error == gorm.ErrRecordNotFound {
		// Create new follow
		follow := models.Follow{
			FollowerUserID:  followerID,
			FollowingUserID: targetUser.ID,
			Status:          "pending",
		}

		if err := tx.Create(&follow).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to follow user"})
			return
		}

		// Create activity log
		activity := models.ActivityLog{
			UserID:    followerID,
			Activity:  "user_followed",
			CreatedAt: time.Now(),
		}

		if err := tx.Create(&activity).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity log"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{
			"following": true,
			"message":   "Successfully followed user",
		})
	} else {
		// Unfollow user
		if err := tx.Delete(&existingFollow).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unfollow user"})
			return
		}

		tx.Commit()
		c.JSON(http.StatusOK, gin.H{
			"following": false,
			"message":   "Successfully unfollowed user",
		})
	}
}

// GetUserFollowers godoc
// @Summary Get user's followers
// @Description Returns paginated list of user's followers
// @Tags interactions
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 20)"
// @Success 200 {object} map[string]interface{}
// @Router /users/{userId}/followers [get]
func (ic *InteractionController) GetUserFollowers(c *gin.Context) {
	userID := c.Param("userId")
	page, _ := c.GetQuery("page")
	pageSize, _ := c.GetQuery("pageSize")

	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "20"
	}

	offset := (convertToInt(page) - 1) * convertToInt(pageSize)

	var followers []struct {
		UserID    uint      `json:"userId"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"followedAt"`
	}

	var total int64
	ic.DB.Model(&models.Follow{}).Where("following_id = ?", userID).Count(&total)

	result := ic.DB.Model(&models.Follow{}).
		Select("users.id as user_id, users.username, follows.created_at").
		Joins("JOIN users ON users.id = follows.follower_id").
		Where("follows.following_id = ?", userID).
		Offset(offset).
		Limit(convertToInt(pageSize)).
		Find(&followers)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching followers"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"followers": followers,
		"pagination": gin.H{
			"currentPage": convertToInt(page),
			"pageSize":    convertToInt(pageSize),
			"totalItems":  total,
			"totalPages":  (total + int64(convertToInt(pageSize)) - 1) / int64(convertToInt(pageSize)),
		},
	})
}

// GetUserFollowing godoc
// @Summary Get users that a user is following
// @Description Returns paginated list of users that the specified user is following
// @Tags interactions
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 20)"
// @Success 200 {object} map[string]interface{}
// @Router /users/{userId}/following [get]
func (ic *InteractionController) GetUserFollowing(c *gin.Context) {
	userID := c.Param("userId")
	page, _ := c.GetQuery("page")
	pageSize, _ := c.GetQuery("pageSize")

	if page == "" {
		page = "1"
	}
	if pageSize == "" {
		pageSize = "20"
	}

	offset := (convertToInt(page) - 1) * convertToInt(pageSize)

	var following []struct {
		UserID    uint      `json:"userId"`
		Username  string    `json:"username"`
		CreatedAt time.Time `json:"followedAt"`
	}

	var total int64
	ic.DB.Model(&models.Follow{}).Where("follower_id = ?", userID).Count(&total)

	result := ic.DB.Model(&models.Follow{}).
		Select("users.id as user_id, users.username, follows.created_at").
		Joins("JOIN users ON users.id = follows.following_id").
		Where("follows.follower_id = ?", userID).
		Offset(offset).
		Limit(convertToInt(pageSize)).
		Find(&following)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching following users"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"following": following,
		"pagination": gin.H{
			"currentPage": convertToInt(page),
			"pageSize":    convertToInt(pageSize),
			"totalItems":  total,
			"totalPages":  (total + int64(convertToInt(pageSize)) - 1) / int64(convertToInt(pageSize)),
		},
	})
}

// Helper function to convert string to int
func convertToInt(str string) int {
	val := 0
	fmt.Sscanf(str, "%d", &val)
	return val
}
