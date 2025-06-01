package controllers

import (
	"math"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type FeedController struct {
	DB *gorm.DB
}

type FeedQuery struct {
	Page         int      `form:"page,default=1" binding:"min=1"`
	PageSize     int      `form:"pageSize,default=20" binding:"min=1,max=50"`
	SortBy       string   `form:"sortBy" binding:"omitempty,oneof=newest popular trending friends_activity"`
	TimeFrame    string   `form:"timeFrame" binding:"omitempty,oneof=today this_week this_month all_time"`
	Latitude     float64  `form:"latitude"`
	Longitude    float64  `form:"longitude"`
	Radius       float64  `form:"radius,default=10" binding:"omitempty,min=0.1,max=100"` // in kilometers
	Categories   []string `form:"categories" binding:"omitempty"`
	Hashtags     []string `form:"hashtags" binding:"omitempty"`
	OnlyFriends  bool     `form:"onlyFriends"`
	NearbyPlaces bool     `form:"nearbyPlaces"`
}

func NewFeedController(db *gorm.DB) *FeedController {
	return &FeedController{DB: db}
}

// GetUserFeed godoc
// @Summary Get user's personalized feed
// @Description Returns posts from followed users and popular posts, with various filtering options
// @Tags feed
// @Accept json
// @Produce json
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 20, max: 50)"
// @Param sortBy query string false "Sort by: newest, popular, trending, friends_activity"
// @Param timeFrame query string false "Time frame: today, this_week, this_month, all_time"
// @Param latitude query number false "User's latitude for location-based feed"
// @Param longitude query number false "User's longitude for location-based feed"
// @Param radius query number false "Search radius in kilometers (default: 10, max: 100)"
// @Param categories query []string false "Filter by place categories"
// @Param hashtags query []string false "Filter by hashtags"
// @Param onlyFriends query boolean false "Show only friends' activities"
// @Param nearbyPlaces query boolean false "Show posts from nearby places"
// @Success 200 {object} map[string]interface{}
// @Router /feed [get]
func (fc *FeedController) GetUserFeed(c *gin.Context) {
	// Get user from context using utils
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}
	userID := user.UserID

	var query FeedQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Base query
	db := fc.DB.Model(&models.Post{})

	// Join necessary tables
	db = db.Joins("JOIN users ON posts.user_id = users.id")
	db = db.Joins("JOIN places ON posts.place_id = places.id")

	// Filter by followed users if not showing only nearby places
	if !query.NearbyPlaces {
		db = db.Joins("JOIN follows ON posts.user_id = follows.following_id").
			Where("follows.follower_id = ?", userID)
	}

	// Apply location-based filtering if coordinates are provided
	if query.Latitude != 0 && query.Longitude != 0 {
		// Haversine formula for distance calculation
		db = db.Where(`
			(6371 * acos(cos(radians(?)) * 
			cos(radians(places.latitude)) * 
			cos(radians(places.longitude) - 
			radians(?)) + 
			sin(radians(?)) * 
			sin(radians(places.latitude)))) <= ?`,
			query.Latitude, query.Longitude, query.Latitude, query.Radius)
	}

	// Apply category filtering
	if len(query.Categories) > 0 {
		db = db.Where("places.categories && ?", query.Categories)
	}

	// Apply hashtag filtering
	if len(query.Hashtags) > 0 {
		hashtagConditions := make([]string, len(query.Hashtags))
		hashtagValues := make([]interface{}, len(query.Hashtags))
		for i, tag := range query.Hashtags {
			hashtagConditions[i] = "posts.content ILIKE ?"
			hashtagValues[i] = "%" + tag + "%"
		}
		db = db.Where(strings.Join(hashtagConditions, " OR "), hashtagValues...)
	}

	// Apply time frame filter
	switch query.TimeFrame {
	case "today":
		db = db.Where("posts.created_at >= CURRENT_DATE")
	case "this_week":
		db = db.Where("posts.created_at >= DATE_TRUNC('week', CURRENT_DATE)")
	case "this_month":
		db = db.Where("posts.created_at >= DATE_TRUNC('month', CURRENT_DATE)")
	}

	// Apply sorting
	switch query.SortBy {
	case "popular":
		db = db.Order("(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) DESC")
	case "trending":
		db = db.Order(`
			(
				SELECT COUNT(*) FROM likes 
				WHERE likes.post_id = posts.id 
				AND likes.created_at >= NOW() - INTERVAL '24 hours'
			) * 3 + 
			(
				SELECT COUNT(*) FROM comments 
				WHERE comments.post_id = posts.id 
				AND comments.created_at >= NOW() - INTERVAL '24 hours'
			) * 2 +
			(
				EXTRACT(EPOCH FROM posts.created_at) / (
					EXTRACT(EPOCH FROM NOW()) - EXTRACT(EPOCH FROM posts.created_at) + 7200
				)
			) DESC
		`)
	case "friends_activity":
		// Posts that friends have interacted with recently
		db = db.Where(`
			EXISTS (
				SELECT 1 FROM likes 
				WHERE likes.post_id = posts.id 
				AND likes.user_id IN (
					SELECT following_id FROM follows WHERE follower_id = ?
				)
				AND likes.created_at >= NOW() - INTERVAL '24 hours'
			)
			OR
			EXISTS (
				SELECT 1 FROM comments
				WHERE comments.post_id = posts.id
				AND comments.user_id IN (
					SELECT following_id FROM follows WHERE follower_id = ?
				)
				AND comments.created_at >= NOW() - INTERVAL '24 hours'
			)
		`, userID, userID).
			Order("posts.created_at DESC")
	default: // "newest" or empty
		db = db.Order("posts.created_at DESC")
	}

	// Calculate pagination
	offset := (query.Page - 1) * query.PageSize
	var total int64
	db.Count(&total)

	// Structure to hold post data with additional information
	var posts []struct {
		models.Post
		Username        string    `json:"username"`
		UserAvatar      string    `json:"userAvatar"`
		LikesCount      int64     `json:"likesCount"`
		CommentsCount   int64     `json:"commentsCount"`
		PlaceName       string    `json:"placeName"`
		PlaceCategories []string  `json:"placeCategories"`
		PlacePointValue int       `json:"placePointValue"`
		Distance        float64   `json:"distance,omitempty"`
		IsLiked         bool      `json:"isLiked"`
		FriendsLiked    []string  `json:"friendsLiked"`
		CreatedAt       time.Time `json:"createdAt"`
	}

	// Get posts with all necessary information including conditional place point_value
	result := db.
		Select(`
			posts.*,
			users.username,
			users.avatar as user_avatar,
			places.name as place_name,
			places.categories as place_categories,
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts p2 WHERE p2.place_id = places.id AND p2.user_id = ?) 
				THEN 1 
				ELSE places.base_points 
			END as place_point_value,
			(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count,
			(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count,
			EXISTS(SELECT 1 FROM likes WHERE likes.post_id = posts.id AND likes.user_id = ?) as is_liked,
			CASE 
				WHEN ? != 0 AND ? != 0 THEN 
					(6371 * acos(cos(radians(?)) * 
					cos(radians(places.latitude)) * 
					cos(radians(places.longitude) - 
					radians(?)) + 
					sin(radians(?)) * 
					sin(radians(places.latitude))))
				ELSE NULL
			END as distance,
			(
				SELECT array_agg(DISTINCT u.username)
				FROM likes l
				JOIN users u ON l.user_id = u.id
				JOIN follows f ON l.user_id = f.following_id
				WHERE l.post_id = posts.id
				AND f.follower_id = ?
				LIMIT 3
			) as friends_liked
		`, userID, userID, query.Latitude, query.Longitude, query.Latitude, query.Longitude, query.Latitude, userID).
		Offset(offset).
		Limit(query.PageSize).
		Find(&posts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching feed"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"pagination": gin.H{
			"currentPage": query.Page,
			"pageSize":    query.PageSize,
			"totalItems":  total,
			"totalPages":  math.Ceil(float64(total) / float64(query.PageSize)),
		},
	})
}
