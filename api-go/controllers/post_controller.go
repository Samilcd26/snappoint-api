package controllers

import (
	"math"
	"net/http"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type PostController struct {
	DB *gorm.DB
}

type CreatePostRequest struct {
	Content   string   `json:"content" binding:"required"`
	MediaType string   `json:"mediaType" binding:"required,oneof=photo video"`
	MediaURLs []string `json:"mediaUrls" binding:"required,min=1"`
	PlaceID   uint     `json:"placeId" binding:"required"`
	Latitude  float64  `json:"latitude" binding:"required"`
	Longitude float64  `json:"longitude" binding:"required"`
	Hashtags  []string `json:"hashtags"`
	Tags      []string `json:"tags"`
	IsPublic  bool     `json:"isPublic"`
	Mood      string   `json:"mood"`
	Weather   string   `json:"weather"`
}

type UpdatePostRequest struct {
	Content   string   `json:"content"`
	MediaURLs []string `json:"mediaUrls"`
	Hashtags  []string `json:"hashtags"`
	Tags      []string `json:"tags"`
	IsPublic  *bool    `json:"isPublic"`
	Mood      string   `json:"mood"`
	Weather   string   `json:"weather"`
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
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
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

	// Extract hashtags from content if not provided
	hashtags := req.Hashtags
	if len(hashtags) == 0 {
		hashtags = extractHashtags(req.Content)
	}

	// Start transaction
	tx := pc.DB.Begin()

	// Create post
	post := models.Post{
		Content:   req.Content,
		MediaType: req.MediaType,
		MediaURL:  req.MediaURLs,
		UserID:    user.UserID,
		PlaceID:   req.PlaceID,
		Latitude:  req.Latitude,
		Longitude: req.Longitude,
		Hashtags:  hashtags,
		Tags:      req.Tags,
		IsPublic:  req.IsPublic,
		Mood:      req.Mood,
		Weather:   req.Weather,
		Season:    getCurrentSeason(),
		Points:    calculateInitialPoints(place.PointValue, req.MediaType),
		CreatedAt: time.Now(),
	}

	if err := tx.Create(&post).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create post"})
		return
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

	// Update place statistics
	if err := tx.Model(&place).Updates(map[string]interface{}{
		"total_visits": gorm.Expr("total_visits + ?", 1),
	}).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update place statistics"})
		return
	}

	// Update user points
	if err := tx.Model(&models.User{}).Where("id = ?", user.UserID).
		Update("total_points", gorm.Expr("total_points + ?", post.Points)).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update user points"})
		return
	}

	// Commit transaction
	if err := tx.Commit().Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to commit transaction"})
		return
	}

	// Return created post with additional info
	var postResponse struct {
		models.Post
		Username     string `json:"username"`
		PlaceName    string `json:"placeName"`
		PointsEarned int    `json:"pointsEarned"`
	}

	pc.DB.Model(&post).
		Select("posts.*, users.username, places.name as place_name").
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		First(&postResponse)

	postResponse.PointsEarned = post.Points

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
		// Re-extract hashtags if content is updated and no explicit hashtags provided
		if len(req.Hashtags) == 0 {
			updates["hashtags"] = extractHashtags(req.Content)
		}
	}
	if len(req.MediaURLs) > 0 {
		updates["media_url"] = req.MediaURLs
	}
	if len(req.Hashtags) > 0 {
		updates["hashtags"] = req.Hashtags
	}
	if len(req.Tags) > 0 {
		updates["tags"] = req.Tags
	}
	if req.IsPublic != nil {
		updates["is_public"] = *req.IsPublic
	}
	if req.Mood != "" {
		updates["mood"] = req.Mood
	}
	if req.Weather != "" {
		updates["weather"] = req.Weather
	}
	updates["updated_at"] = time.Now()

	// Update post
	if err := tx.Model(&post).Updates(updates).Error; err != nil {
		tx.Rollback()
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update post"})
		return
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
	var postResponse struct {
		models.Post
		Username  string `json:"username"`
		PlaceName string `json:"placeName"`
	}

	pc.DB.Model(&post).
		Select("posts.*, users.username, places.name as place_name").
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		First(&postResponse)

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

	// Update user points (subtract post points)
	if err := tx.Model(&models.User{}).Where("id = ?", userID).
		Update("total_points", gorm.Expr("total_points - ?", post.Points)).Error; err != nil {
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
		"points_deducted": post.Points,
	})
}

// GetUserPosts godoc
// @Summary Get posts by user
// @Description Returns paginated list of posts by a specific user
// @Tags posts
// @Accept json
// @Produce json
// @Param userId path string true "User ID"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 20)"
// @Success 200 {object} map[string]interface{}
// @Router /users/{userId}/posts [get]
func (pc *PostController) GetUserPosts(c *gin.Context) {
	userID := c.Param("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))

	offset := (page - 1) * pageSize

	var posts []struct {
		models.Post
		Username      string `json:"username"`
		PlaceName     string `json:"placeName"`
		LikesCount    int64  `json:"likesCount"`
		CommentsCount int64  `json:"commentsCount"`
	}

	var total int64
	pc.DB.Model(&models.Post{}).Where("user_id = ?", userID).Count(&total)

	result := pc.DB.Model(&models.Post{}).
		Select(`
			posts.*,
			users.username,
			places.name as place_name,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count
		`).
		Joins("JOIN users ON posts.user_id = users.id").
		Joins("JOIN places ON posts.place_id = places.id").
		Where("posts.user_id = ?", userID).
		Order("posts.created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&posts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"pagination": gin.H{
			"currentPage": page,
			"pageSize":    pageSize,
			"totalItems":  total,
			"totalPages":  math.Ceil(float64(total) / float64(pageSize)),
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

// Helper function to extract hashtags from content
func extractHashtags(content string) []string {
	words := strings.Fields(content)
	var hashtags []string
	for _, word := range words {
		if strings.HasPrefix(word, "#") {
			hashtag := strings.TrimPrefix(word, "#")
			if hashtag != "" {
				hashtags = append(hashtags, hashtag)
			}
		}
	}
	return hashtags
}

// Helper function to get current season
func getCurrentSeason() string {
	month := time.Now().Month()
	switch {
	case month >= 3 && month <= 5:
		return "spring"
	case month >= 6 && month <= 8:
		return "summer"
	case month >= 9 && month <= 11:
		return "autumn"
	default:
		return "winter"
	}
}

// Helper function to calculate initial points for a post
func calculateInitialPoints(placePointValue int, mediaType string) int {
	basePoints := placePointValue

	// Bonus points for media type
	switch mediaType {
	case "video":
		basePoints += 5 // Extra points for video content
	case "photo":
		basePoints += 2 // Extra points for photo content
	}

	// Time-based bonus (e.g., peak hours)
	hour := time.Now().Hour()
	if hour >= 11 && hour <= 14 || hour >= 18 && hour <= 21 {
		basePoints += 3 // Bonus points during peak hours
	}

	return basePoints
}
