package controllers

import (
	"fmt"
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type PostController struct {
	DB *gorm.DB
}

// Common response structures
type PostUser struct {
	ID          uint   `json:"id"`
	Username    string `json:"username"`
	FirstName   string `json:"firstName"`
	LastName    string `json:"lastName"`
	Avatar      string `json:"avatar"`
	TotalPoints int64  `json:"totalPoints,omitempty"`
}

type PostPlace struct {
	ID         uint   `json:"id"`
	Name       string `json:"name"`
	Address    string `json:"address,omitempty"`
	PointValue int    `json:"pointValue,omitempty"`
	Image      string `json:"image,omitempty"`
}

type PostMediaItem struct {
	ID         uint     `json:"id"`
	MediaType  string   `json:"mediaType"`
	MediaURL   string   `json:"mediaUrl"`
	OrderIndex int      `json:"orderIndex"`
	AltText    string   `json:"altText"`
	Width      int      `json:"width"`
	Height     int      `json:"height"`
	Duration   int      `json:"duration"`
	Tags       []string `json:"tags"`
}

type PostInteraction struct {
	LikesCount    int64 `json:"likesCount"`
	CommentsCount int64 `json:"commentsCount"`
	IsLiked       bool  `json:"isLiked"`
}

type PostSummary struct {
	ID            uint            `json:"id"`
	Caption       string          `json:"caption"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	Latitude      float64         `json:"latitude"`
	Longitude     float64         `json:"longitude"`
	EarnedPoints  int64           `json:"earnedPoints,omitempty"`
	ThumbnailURL  string          `json:"thumbnailUrl"`
	MediaType     string          `json:"mediaType"`
	MediaCount    int64           `json:"mediaCount"`
	User          PostUser        `json:"user"`
	Place         PostPlace       `json:"place"`
	Interaction   PostInteraction `json:"interaction"`
}

type PostDetail struct {
	ID            uint            `json:"id"`
	Caption       string          `json:"caption"`
	CreatedAt     time.Time       `json:"createdAt"`
	UpdatedAt     time.Time       `json:"updatedAt"`
	Latitude      float64         `json:"latitude"`
	Longitude     float64         `json:"longitude"`
	EarnedPoints  int64           `json:"earnedPoints"`
	IsPublic      bool            `json:"isPublic"`
	AllowComments bool            `json:"allowComments"`
	User          PostUser        `json:"user"`
	Place         PostPlace       `json:"place"`
	MediaItems    []PostMediaItem `json:"mediaItems"`
	Interaction   PostInteraction `json:"interaction"`
	RecentLikes   []PostUser      `json:"recentLikes"`
	RecentComments []struct {
		ID        uint      `json:"id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"createdAt"`
		User      PostUser  `json:"user"`
	} `json:"recentComments"`
}

type CreatePostRequest struct {
	PostCaption string `json:"postCaption" binding:"omitempty"`
	MediaItems  []struct {
		MediaType string   `json:"mediaType" binding:"required,oneof=photo video"`
		MediaURL  string   `json:"mediaUrl" binding:"required"`
		Width     int      `json:"width"`
		Height    int      `json:"height"`
		Duration  int      `json:"duration"`
		AltText   string   `json:"altText"`
		Tags      []string `json:"tags"`
	} `json:"mediaItems" binding:"required,dive"`
	PlaceID       uint    `json:"placeId" binding:"required"`
	Latitude      float64 `json:"latitude" binding:"required"`
	Longitude     float64 `json:"longitude" binding:"required"`
	IsPublic      bool    `json:"isPublic" default:"true"`
	AllowComments bool    `json:"allowComments" default:"true"`
}

type UpdatePostRequest struct {
	Content    string `json:"content"`
	MediaItems []struct {
		MediaID    uint     `json:"mediaId,omitempty"`
		MediaType  string   `json:"mediaType" binding:"omitempty,oneof=photo video"`
		MediaURL   string   `json:"mediaUrl"`
		Width      int      `json:"width"`
		Height     int      `json:"height"`
		Duration   int      `json:"duration"`
		AltText    string   `json:"altText"`
		OrderIndex int      `json:"orderIndex"`
		Tags       []string `json:"tags"`
	} `json:"mediaItems"`
	IsPublic      *bool `json:"isPublic"`
	AllowComments *bool `json:"allowComments"`
}

func NewPostController(db *gorm.DB) *PostController {
	return &PostController{DB: db}
}

// CreatePost godoc
// @Summary Create a new post
// @Description Creates a new post with location verification and awards points
// @Tags posts
// @Accept json
// @Produce json
// @Param post body CreatePostRequest true "Post creation request"
// @Success 201 {object} models.Post
// @Router /posts [post]
func (pc *PostController) CreatePost(c *gin.Context) {
	user := utils.GetUser(c)
	var req CreatePostRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		fmt.Println(err, "burda err var")
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate that at least one media item is provided
	if len(req.MediaItems) == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "At least one media item is required"})
		return
	}

	// Get place details
	var place models.Place
	if err := pc.DB.First(&place, req.PlaceID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Verify user's location is near the place
	distance := calculateDistance(
		req.Latitude, req.Longitude,
		place.Latitude, place.Longitude,
	)

	// Maximum allowed distance in meters (e.g., 100 meters)
	const maxDistance = 100.0
	if distance > maxDistance {
		c.JSON(http.StatusBadRequest, gin.H{
			"error": "You must be at the location to create a post",
			"distance": gin.H{
				"current": distance,
				"maximum": maxDistance,
			},
		})
		return
	}

	// Start transaction
	tx := pc.DB.Begin()

	// Create post
	earnedPoints := calculateInitialPoints(place.BasePoints, req.MediaItems[0].MediaType)
	post := models.Post{
		PostCaption:   req.PostCaption,
		UserID:        user.UserID,
		PlaceID:       req.PlaceID,
		Latitude:      req.Latitude,
		Longitude:     req.Longitude,
		IsPublic:      req.IsPublic,
		AllowComments: req.AllowComments,
		EarnedPoints:  earnedPoints,
		CreatedAt:     time.Now(),
	}

	if err := tx.Create(&post).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
	}

	// Create media items
	for i, mediaItem := range req.MediaItems {
		postMedia := models.PostMedia{
			PostID:     post.ID,
			MediaType:  mediaItem.MediaType,
			MediaURL:   mediaItem.MediaURL,
			OrderIndex: i,
			AltText:    mediaItem.AltText,
			Width:      mediaItem.Width,
			Height:     mediaItem.Height,
			Duration:   mediaItem.Duration,
			Tags:       mediaItem.Tags,
		}

		if err := tx.Create(&postMedia).Error; err != nil {
			tx.Rollback()
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create media items"})
			return
		}
	}

	// Create activity log
	activity := models.ActivityLog{
		UserID:    user.UserID,
		PlaceID:   req.PlaceID,
		PostID:    post.ID,
		Activity:  "post_created",
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		CreatedAt: time.Now(),
	}

	if err := tx.Create(&activity).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity log"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return created post with additional info
	type PostResponse struct {
		models.Post
		Username     string             `json:"username"`
		PlaceName    string             `json:"placeName"`
		PointsEarned int64              `json:"pointsEarned"`
		MediaItems   []models.PostMedia `json:"mediaItems" gorm:"foreignKey:PostID"`
	}

	var postResponse PostResponse

	pc.DB.Model(&post).
		Select("posts.*, users.username, places.name as place_name").
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		First(&postResponse)

	// Get media items
	pc.DB.Model(&models.PostMedia{}).
		Where("post_id = ?", post.ID).
		Order("order_index").
		Find(&postResponse.MediaItems)

	postResponse.PointsEarned = earnedPoints

	c.JSON(http.StatusCreated, postResponse)
}

// UpdatePost godoc
// @Summary Update an existing post
// @Description Updates post content and metadata
// @Tags posts
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Param post body UpdatePostRequest true "Post update request"
// @Success 200 {object} models.Post
// @Router /posts/{id} [put]
func (pc *PostController) UpdatePost(c *gin.Context) {
	userID := c.GetUint("userID")
	postID := c.Param("id")
	var req UpdatePostRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Get existing post
	var post models.Post
	if err := pc.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Verify ownership
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only update your own posts"})
		return
	}

	// Start transaction
	tx := pc.DB.Begin()

	// Update post fields if provided
	updates := make(map[string]interface{})

	if req.Content != "" {
		updates["content"] = req.Content
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.AllowComments != nil {
		updates["allow_comments"] = *req.AllowComments
	}
	updates["updated_at"] = time.Now()

	// Update post
	if err := tx.Model(&post).Updates(updates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
	}

	// Handle media updates if provided
	if len(req.MediaItems) > 0 {
		// Delete existing media items that are not in the update request
		existingMediaIDs := make([]uint, 0)
		for _, media := range req.MediaItems {
			if media.MediaID != 0 {
				existingMediaIDs = append(existingMediaIDs, media.MediaID)
			}
		}

		if len(existingMediaIDs) > 0 {
			if err := tx.Where("post_id = ? AND media_id NOT IN ?", post.ID, existingMediaIDs).
				Delete(&models.PostMedia{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media items"})
				return
			}
		} else {
			// If no existing media IDs provided, delete all media items
			if err := tx.Where("post_id = ?", post.ID).Delete(&models.PostMedia{}).Error; err != nil {
				tx.Rollback()
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media items"})
				return
			}
		}

		// Create or update media items
		for _, mediaItem := range req.MediaItems {
			var postMedia models.PostMedia
			if mediaItem.MediaID != 0 {
				// Update existing media item
				if err := tx.First(&postMedia, mediaItem.MediaID).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusNotFound, gin.H{"error": "Media item not found"})
					return
				}

				postMedia.MediaType = mediaItem.MediaType
				postMedia.MediaURL = mediaItem.MediaURL
				postMedia.OrderIndex = mediaItem.OrderIndex
				postMedia.AltText = mediaItem.AltText
				postMedia.Width = mediaItem.Width
				postMedia.Height = mediaItem.Height
				postMedia.Duration = mediaItem.Duration
				postMedia.Tags = mediaItem.Tags

				if err := tx.Save(&postMedia).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update media item"})
					return
				}
			} else {
				// Create new media item
				postMedia = models.PostMedia{
					PostID:     post.ID,
					MediaType:  mediaItem.MediaType,
					MediaURL:   mediaItem.MediaURL,
					OrderIndex: mediaItem.OrderIndex,
					AltText:    mediaItem.AltText,
					Width:      mediaItem.Width,
					Height:     mediaItem.Height,
					Duration:   mediaItem.Duration,
					Tags:       mediaItem.Tags,
				}

				if err := tx.Create(&postMedia).Error; err != nil {
					tx.Rollback()
					c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create media item"})
					return
				}
			}
		}
	}

	// Create activity log
	activity := models.ActivityLog{
		UserID:    userID,
		PlaceID:   post.PlaceID,
		PostID:    post.ID,
		Activity:  "post_updated",
		CreatedAt: time.Now(),
	}

	if err := tx.Create(&activity).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity log"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return updated post with additional info
	type UpdatePostResponse struct {
		models.Post
		Username   string             `json:"username"`
		PlaceName  string             `json:"placeName"`
		MediaItems []models.PostMedia `json:"mediaItems" gorm:"foreignKey:PostID"`
	}

	var postResponse UpdatePostResponse

	pc.DB.Model(&post).
		Select("posts.*, users.username, places.name as place_name").
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		First(&postResponse)

	// Get media items
	pc.DB.Model(&models.PostMedia{}).
		Where("post_id = ?", post.ID).
		Order("order_index").
		Find(&postResponse.MediaItems)

	c.JSON(http.StatusOK, postResponse)
}

// DeletePost godoc
// @Summary Delete a post
// @Description Deletes a post and related data
// @Tags posts
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} map[string]interface{}
// @Router /posts/{id} [delete]
func (pc *PostController) DeletePost(c *gin.Context) {
	userID := c.GetUint("userID")
	postID := c.Param("id")

	// Get existing post
	var post models.Post
	if err := pc.DB.First(&post, postID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Post not found"})
		return
	}

	// Verify ownership
	if post.UserID != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "You can only delete your own posts"})
		return
	}

	// Start transaction
	tx := pc.DB.Begin()

	// Delete media items
	if err := tx.Where("post_id = ?", postID).Delete(&models.PostMedia{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete media items"})
		return
	}

	// Delete likes
	if err := tx.Where("post_id = ?", postID).Delete(&models.Like{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete likes"})
		return
	}

	// Delete comments
	if err := tx.Where("post_id = ?", postID).Delete(&models.Comment{}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete comments"})
		return
	}

	// Create activity log before deleting post
	activity := models.ActivityLog{
		UserID:    userID,
		PlaceID:   post.PlaceID,
		Activity:  "post_deleted",
		CreatedAt: time.Now(),
	}

	if err := tx.Create(&activity).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create activity log"})
		return
	}

	// Update user points (subtract earned points)
	if err := tx.Model(&models.User{}).Where("id = ?", userID).
		Update("total_points", gorm.Expr("total_points - ?", post.EarnedPoints)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user points"})
		return
	}

	// Delete post
	if err := tx.Delete(&post).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete post"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message":         "Post successfully deleted",
		"points_deducted": post.EarnedPoints,
	})
}

// GetUserPosts godoc
// @Summary Get posts by user (summary view)
// @Description Returns paginated list of posts by a specific user with minimal info for grid view
// @Tags posts
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 30)"
// @Success 200 {object} StandardResponse
// @Router /users/{userId}/posts [get]
func (pc *PostController) GetUserPosts(c *gin.Context) {
	userID := c.Param("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))

	offset := (page - 1) * pageSize

	// Count total posts
	var total int64
	pc.DB.Model(&models.Post{}).Where("user_id = ?", userID).Count(&total)

	// Get posts data
	var rawPosts []struct {
		ID           uint      `gorm:"column:id"`
		Caption      string    `gorm:"column:post_caption"`
		CreatedAt    time.Time `gorm:"column:created_at"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
		Latitude     float64   `gorm:"column:latitude"`
		Longitude    float64   `gorm:"column:longitude"`
		EarnedPoints int64     `gorm:"column:earned_points"`
		PlaceID      uint      `gorm:"column:place_id"`
		PlaceName    string    `gorm:"column:place_name"`
		UserID       uint      `gorm:"column:user_id"`
		Username     string    `gorm:"column:username"`
		FirstName    string    `gorm:"column:first_name"`
		LastName     string    `gorm:"column:last_name"`
		Avatar       string    `gorm:"column:avatar"`
		LikesCount   int64     `gorm:"column:likes_count"`
		CommentsCount int64    `gorm:"column:comments_count"`
		ThumbnailURL string    `gorm:"column:thumbnail_url"`
		MediaType    string    `gorm:"column:media_type"`
		MediaCount   int64     `gorm:"column:media_count"`
	}

	result := pc.DB.Model(&models.Post{}).
		Select(`
			posts.id,
			posts.post_caption,
			posts.created_at,
			posts.updated_at,
			posts.latitude,
			posts.longitude,
			posts.earned_points,
			posts.place_id,
			places.name as place_name,
			posts.user_id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count,
			(SELECT media_url FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as thumbnail_url,
			(SELECT media_type FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as media_type,
			(SELECT COUNT(*) FROM post_media WHERE post_media.post_id = posts.id) as media_count
		`).
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		Where("posts.user_id = ?", userID).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&rawPosts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Message: "Error fetching posts",
		})
		return
	}

	// Transform to standard format
	posts := make([]PostSummary, len(rawPosts))
	for i, raw := range rawPosts {
		posts[i] = PostSummary{
			ID:           raw.ID,
			Caption:      raw.Caption,
			CreatedAt:    raw.CreatedAt,
			UpdatedAt:    raw.UpdatedAt,
			Latitude:     raw.Latitude,
			Longitude:    raw.Longitude,
			EarnedPoints: raw.EarnedPoints,
			ThumbnailURL: raw.ThumbnailURL,
			MediaType:    raw.MediaType,
			MediaCount:   raw.MediaCount,
			User: PostUser{
				ID:        raw.UserID,
				Username:  raw.Username,
				FirstName: raw.FirstName,
				LastName:  raw.LastName,
				Avatar:    raw.Avatar,
			},
			Place: PostPlace{
				ID:   raw.PlaceID,
				Name: raw.PlaceName,
			},
			Interaction: PostInteraction{
				LikesCount:    raw.LikesCount,
				CommentsCount: raw.CommentsCount,
			},
		}
	}

	// Standard response
	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    posts,
		Pagination: &PaginationMeta{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  total,
			TotalPages:  int(math.Ceil(float64(total) / float64(pageSize))),
		},
	})
}

// GetPostDetail godoc
// @Summary Get detailed information about a specific post
// @Description Returns comprehensive post information including user, place, media, likes, and comments
// @Tags posts
// @Accept json
// @Produce json
// @Param id path string true "Post ID"
// @Success 200 {object} StandardResponse
// @Router /posts/{id} [get]
func (pc *PostController) GetPostDetail(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Message: "User not found in context",
		})
		return
	}

	postID := c.Param("id")

	// Get post with all related information
	var rawPost struct {
		ID              uint      `gorm:"column:id"`
		Caption         string    `gorm:"column:post_caption"`
		CreatedAt       time.Time `gorm:"column:created_at"`
		UpdatedAt       time.Time `gorm:"column:updated_at"`
		Latitude        float64   `gorm:"column:latitude"`
		Longitude       float64   `gorm:"column:longitude"`
		EarnedPoints    int64     `gorm:"column:earned_points"`
		IsPublic        bool      `gorm:"column:is_public"`
		AllowComments   bool      `gorm:"column:allow_comments"`
		UserID          uint      `gorm:"column:user_id"`
		Username        string    `gorm:"column:username"`
		UserFirstName   string    `gorm:"column:user_first_name"`
		UserLastName    string    `gorm:"column:user_last_name"`
		UserAvatar      string    `gorm:"column:user_avatar"`
		UserTotalPoints int64     `gorm:"column:user_total_points"`
		PlaceID         uint      `gorm:"column:place_id"`
		PlaceName       string    `gorm:"column:place_name"`
		PlaceAddress    string    `gorm:"column:place_address"`
		PlacePointValue int       `gorm:"column:place_point_value"`
		PlaceImage      string    `gorm:"column:place_image"`
		LikesCount      int64     `gorm:"column:likes_count"`
		CommentsCount   int64     `gorm:"column:comments_count"`
		IsLiked         bool      `gorm:"column:is_liked"`
	}

	result := pc.DB.Model(&models.Post{}).
		Select(`
			posts.id,
			posts.post_caption,
			posts.created_at,
			posts.updated_at,
			posts.latitude,
			posts.longitude,
			posts.earned_points,
			posts.is_public,
			posts.allow_comments,
			posts.user_id,
			users.username,
			users.first_name as user_first_name,
			users.last_name as user_last_name,
			users.avatar as user_avatar,
			users.total_points as user_total_points,
			posts.place_id,
			places.name as place_name,
			places.address as place_address,
			places.place_image as place_image,
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts p2 WHERE p2.place_id = places.id AND p2.user_id = ?) 
				THEN 1 
				ELSE places.base_points 
			END as place_point_value,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count,
			EXISTS(SELECT 1 FROM likes WHERE likes.post_id = posts.id AND likes.user_id = ?) as is_liked
		`, user.UserID, user.UserID).
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		Where("posts.id = ?", postID).
		First(&rawPost)

	if result.Error != nil {
		if result.Error == gorm.ErrRecordNotFound {
			c.JSON(http.StatusNotFound, StandardResponse{
				Success: false,
				Message: "Post not found",
			})
		} else {
			c.JSON(http.StatusInternalServerError, StandardResponse{
				Success: false,
				Message: "Error fetching post",
			})
		}
		return
	}

	// Get media items
	var rawMediaItems []models.PostMedia
	pc.DB.Where("post_id = ?", postID).Order("order_index").Find(&rawMediaItems)

	// Transform media items
	mediaItems := make([]PostMediaItem, len(rawMediaItems))
	for i, media := range rawMediaItems {
		mediaItems[i] = PostMediaItem{
			ID:         media.ID,
			MediaType:  media.MediaType,
			MediaURL:   media.MediaURL,
			OrderIndex: media.OrderIndex,
			AltText:    media.AltText,
			Width:      media.Width,
			Height:     media.Height,
			Duration:   media.Duration,
			Tags:       media.Tags,
		}
	}

	// Get recent likes (last 10)
	var rawRecentLikes []struct {
		UserID    uint   `gorm:"column:user_id"`
		Username  string `gorm:"column:username"`
		FirstName string `gorm:"column:first_name"`
		LastName  string `gorm:"column:last_name"`
		Avatar    string `gorm:"column:avatar"`
	}
	pc.DB.Table("likes").
		Select("users.id as user_id, users.username, users.first_name, users.last_name, users.avatar").
		Joins("JOIN users ON users.id = likes.user_id").
		Where("likes.post_id = ?", postID).
		Order("likes.created_at DESC").
		Limit(10).
		Find(&rawRecentLikes)

	// Transform recent likes
	recentLikes := make([]PostUser, len(rawRecentLikes))
	for i, like := range rawRecentLikes {
		recentLikes[i] = PostUser{
			ID:        like.UserID,
			Username:  like.Username,
			FirstName: like.FirstName,
			LastName:  like.LastName,
			Avatar:    like.Avatar,
		}
	}

	// Get recent comments (last 20)
	var rawRecentComments []struct {
		ID        uint      `gorm:"column:comment_id"`
		Content   string    `gorm:"column:text_content"`
		CreatedAt time.Time `gorm:"column:created_at"`
		UserID    uint      `gorm:"column:user_id"`
		Username  string    `gorm:"column:username"`
		FirstName string    `gorm:"column:first_name"`
		LastName  string    `gorm:"column:last_name"`
		Avatar    string    `gorm:"column:avatar"`
	}
	pc.DB.Table("comments").
		Select("comments.comment_id, comments.text_content, comments.created_at, users.id as user_id, users.username, users.first_name, users.last_name, users.avatar").
		Joins("JOIN users ON users.id = comments.user_id").
		Where("comments.post_id = ?", postID).
		Order("comments.created_at DESC").
		Limit(20).
		Find(&rawRecentComments)

	// Transform recent comments
	recentComments := make([]struct {
		ID        uint      `json:"id"`
		Content   string    `json:"content"`
		CreatedAt time.Time `json:"createdAt"`
		User      PostUser  `json:"user"`
	}, len(rawRecentComments))
	for i, comment := range rawRecentComments {
		recentComments[i] = struct {
			ID        uint      `json:"id"`
			Content   string    `json:"content"`
			CreatedAt time.Time `json:"createdAt"`
			User      PostUser  `json:"user"`
		}{
			ID:        comment.ID,
			Content:   comment.Content,
			CreatedAt: comment.CreatedAt,
			User: PostUser{
				ID:        comment.UserID,
				Username:  comment.Username,
				FirstName: comment.FirstName,
				LastName:  comment.LastName,
				Avatar:    comment.Avatar,
			},
		}
	}

	// Build standard response
	postDetail := PostDetail{
		ID:            rawPost.ID,
		Caption:       rawPost.Caption,
		CreatedAt:     rawPost.CreatedAt,
		UpdatedAt:     rawPost.UpdatedAt,
		Latitude:      rawPost.Latitude,
		Longitude:     rawPost.Longitude,
		EarnedPoints:  rawPost.EarnedPoints,
		IsPublic:      rawPost.IsPublic,
		AllowComments: rawPost.AllowComments,
		User: PostUser{
			ID:          rawPost.UserID,
			Username:    rawPost.Username,
			FirstName:   rawPost.UserFirstName,
			LastName:    rawPost.UserLastName,
			Avatar:      rawPost.UserAvatar,
			TotalPoints: rawPost.UserTotalPoints,
		},
		Place: PostPlace{
			ID:         rawPost.PlaceID,
			Name:       rawPost.PlaceName,
			Address:    rawPost.PlaceAddress,
			PointValue: rawPost.PlacePointValue,
			Image:      rawPost.PlaceImage,
		},
		MediaItems: mediaItems,
		Interaction: PostInteraction{
			LikesCount:    rawPost.LikesCount,
			CommentsCount: rawPost.CommentsCount,
			IsLiked:       rawPost.IsLiked,
		},
		RecentLikes:    recentLikes,
		RecentComments: recentComments,
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    postDetail,
	})
}

// GetUserPostsAtPlace godoc
// @Summary Get all posts by a specific user at a specific place (summary view)
// @Description Returns paginated posts by a user at a specific place with minimal info for grid view
// @Tags posts
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param placeId path string true "Place ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 30)"
// @Success 200 {object} StandardResponse
// @Router /users/{userId}/places/{placeId}/posts [get]
func (pc *PostController) GetUserPostsAtPlace(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Message: "User not found in context",
		})
		return
	}

	userID := c.Param("userId")
	placeID := c.Param("placeId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))

	offset := (page - 1) * pageSize

	// Get user info
	var userInfo PostUser
	if err := pc.DB.Model(&models.User{}).
		Select("id, username, first_name, last_name, avatar").
		Where("id = ?", userID).
		First(&userInfo).Error; err != nil {
		c.JSON(http.StatusNotFound, StandardResponse{
			Success: false,
			Message: "User not found",
		})
		return
	}

	// Get place info
	var placeInfo PostPlace
	if err := pc.DB.Model(&models.Place{}).
		Select(`
			id, name, address, place_image as image,
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
				THEN 1 
				ELSE base_points 
			END as point_value
		`, currentUser.UserID).
		Where("id = ?", placeID).
		First(&placeInfo).Error; err != nil {
		c.JSON(http.StatusNotFound, StandardResponse{
			Success: false,
			Message: "Place not found",
		})
		return
	}

	// Count total posts
	var totalPosts int64
	pc.DB.Model(&models.Post{}).
		Where("user_id = ? AND place_id = ?", userID, placeID).
		Count(&totalPosts)

	if totalPosts == 0 {
		c.JSON(http.StatusOK, StandardResponse{
			Success: true,
			Data:    []PostSummary{},
			Meta: gin.H{
				"user":  userInfo,
				"place": placeInfo,
				"summary": gin.H{
					"totalPosts":  0,
					"totalPoints": 0,
				},
			},
			Pagination: &PaginationMeta{
				CurrentPage: page,
				PageSize:    pageSize,
				TotalItems:  totalPosts,
				TotalPages:  0,
			},
		})
		return
	}

	// Get posts data
	var rawPosts []struct {
		ID           uint      `gorm:"column:id"`
		Caption      string    `gorm:"column:post_caption"`
		CreatedAt    time.Time `gorm:"column:created_at"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
		Latitude     float64   `gorm:"column:latitude"`
		Longitude    float64   `gorm:"column:longitude"`
		EarnedPoints int64     `gorm:"column:earned_points"`
		LikesCount   int64     `gorm:"column:likes_count"`
		CommentsCount int64    `gorm:"column:comments_count"`
		ThumbnailURL string    `gorm:"column:thumbnail_url"`
		MediaType    string    `gorm:"column:media_type"`
		MediaCount   int64     `gorm:"column:media_count"`
		IsLiked      bool      `gorm:"column:is_liked"`
	}

	result := pc.DB.Model(&models.Post{}).
		Select(`
			posts.id,
			posts.post_caption,
			posts.created_at,
			posts.updated_at,
			posts.latitude,
			posts.longitude,
			posts.earned_points,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count,
			(SELECT media_url FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as thumbnail_url,
			(SELECT media_type FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as media_type,
			(SELECT COUNT(*) FROM post_media WHERE post_media.post_id = posts.id) as media_count,
			EXISTS(SELECT 1 FROM likes WHERE likes.post_id = posts.id AND likes.user_id = ?) as is_liked
		`, currentUser.UserID).
		Where("posts.user_id = ? AND posts.place_id = ?", userID, placeID).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&rawPosts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Message: "Error fetching posts",
		})
		return
	}

	// Transform to standard format
	posts := make([]PostSummary, len(rawPosts))
	for i, raw := range rawPosts {
		posts[i] = PostSummary{
			ID:           raw.ID,
			Caption:      raw.Caption,
			CreatedAt:    raw.CreatedAt,
			UpdatedAt:    raw.UpdatedAt,
			Latitude:     raw.Latitude,
			Longitude:    raw.Longitude,
			EarnedPoints: raw.EarnedPoints,
			ThumbnailURL: raw.ThumbnailURL,
			MediaType:    raw.MediaType,
			MediaCount:   raw.MediaCount,
			User:         userInfo,
			Place:        placeInfo,
			Interaction: PostInteraction{
				LikesCount:    raw.LikesCount,
				CommentsCount: raw.CommentsCount,
				IsLiked:       raw.IsLiked,
			},
		}
	}

	// Get summary statistics
	var summary struct {
		TotalPosts  int64 `gorm:"column:total_posts"`
		TotalPoints int64 `gorm:"column:total_points"`
	}
	pc.DB.Model(&models.Post{}).
		Select(`
			COUNT(*) as total_posts,
			COALESCE(SUM(earned_points), 0) as total_points
		`).
		Where("user_id = ? AND place_id = ?", userID, placeID).
		Scan(&summary)

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    posts,
		Meta: gin.H{
			"user":  userInfo,
			"place": placeInfo,
			"summary": gin.H{
				"totalPosts":  summary.TotalPosts,
				"totalPoints": summary.TotalPoints,
			},
		},
		Pagination: &PaginationMeta{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  totalPosts,
			TotalPages:  int(math.Ceil(float64(totalPosts) / float64(pageSize))),
		},
	})
}

// GetPlacePostsGrid godoc
// @Summary Get posts at a place in grid format (Instagram-like)
// @Description Returns posts at a specific place in a grid format with minimal info for gallery view
// @Tags posts
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 30)"
// @Success 200 {object} StandardResponse
// @Router /places/{placeId}/posts/grid [get]
func (pc *PostController) GetPlacePostsGrid(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, StandardResponse{
			Success: false,
			Message: "User not found in context",
		})
		return
	}

	placeID := c.Param("placeId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "30"))

	offset := (page - 1) * pageSize

	// Get place info
	var place PostPlace
	if err := pc.DB.Model(&models.Place{}).
		Select("id, name").
		Where("id = ?", placeID).
		First(&place).Error; err != nil {
		c.JSON(http.StatusNotFound, StandardResponse{
			Success: false,
			Message: "Place not found",
		})
		return
	}

	// Count total posts
	var totalPosts int64
	pc.DB.Model(&models.Post{}).Where("place_id = ?", placeID).Count(&totalPosts)

	// Get grid posts data
	var rawPosts []struct {
		ID           uint    `gorm:"column:id"`
		UserID       uint    `gorm:"column:user_id"`
		Username     string  `gorm:"column:username"`
		FirstName    string  `gorm:"column:first_name"`
		LastName     string  `gorm:"column:last_name"`
		Avatar       string  `gorm:"column:avatar"`
		Latitude     float64 `gorm:"column:latitude"`
		Longitude    float64 `gorm:"column:longitude"`
		ThumbnailURL string  `gorm:"column:thumbnail_url"`
		MediaType    string  `gorm:"column:media_type"`
		MediaCount   int64   `gorm:"column:media_count"`
		LikesCount   int64   `gorm:"column:likes_count"`
		CreatedAt    time.Time `gorm:"column:created_at"`
		UpdatedAt    time.Time `gorm:"column:updated_at"`
	}

	result := pc.DB.Model(&models.Post{}).
		Select(`
			posts.id,
			posts.user_id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			posts.latitude,
			posts.longitude,
			posts.created_at,
			posts.updated_at,
			(SELECT media_url FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as thumbnail_url,
			(SELECT media_type FROM post_media WHERE post_media.post_id = posts.id ORDER BY order_index LIMIT 1) as media_type,
			(SELECT COUNT(*) FROM post_media WHERE post_media.post_id = posts.id) as media_count,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count
		`).
		Joins("JOIN users ON posts.user_id = users.id").
		Where("posts.place_id = ?", placeID).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&rawPosts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, StandardResponse{
			Success: false,
			Message: "Error fetching posts",
		})
		return
	}

	// Transform to standard format
	posts := make([]PostSummary, len(rawPosts))
	for i, raw := range rawPosts {
		posts[i] = PostSummary{
			ID:           raw.ID,
			CreatedAt:    raw.CreatedAt,
			UpdatedAt:    raw.UpdatedAt,
			Latitude:     raw.Latitude,
			Longitude:    raw.Longitude,
			ThumbnailURL: raw.ThumbnailURL,
			MediaType:    raw.MediaType,
			MediaCount:   raw.MediaCount,
			User: PostUser{
				ID:        raw.UserID,
				Username:  raw.Username,
				FirstName: raw.FirstName,
				LastName:  raw.LastName,
				Avatar:    raw.Avatar,
			},
			Place: place,
			Interaction: PostInteraction{
				LikesCount: raw.LikesCount,
			},
		}
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    posts,
		Meta: gin.H{
			"place": place,
		},
		Pagination: &PaginationMeta{
			CurrentPage: page,
			PageSize:    pageSize,
			TotalItems:  totalPosts,
			TotalPages:  int(math.Ceil(float64(totalPosts) / float64(pageSize))),
		},
	})
}

// Helper function to calculate distance between two points using Haversine formula
func calculateDistance(lat1, lon1, lat2, lon2 float64) float64 {
	const R = 6371000 // Earth's radius in meters

	φ1 := lat1 * math.Pi / 180
	φ2 := lat2 * math.Pi / 180
	Δφ := (lat2 - lat1) * math.Pi / 180
	Δλ := (lon2 - lon1) * math.Pi / 180

	a := math.Sin(Δφ/2)*math.Sin(Δφ/2) +
		math.Cos(φ1)*math.Cos(φ2)*
			math.Sin(Δλ/2)*math.Sin(Δλ/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))

	return R * c // Distance in meters
}

// Helper function to calculate initial points for a post
func calculateInitialPoints(placePointValue int, mediaType string) int64 {
	basePoints := placePointValue

	// Bonus points for media type
	switch mediaType {
	case "video":
		basePoints += 5 // Extra points for video content
	case "photo":
		basePoints += 2 // Extra points for photo content
	}

	return int64(basePoints)
}
