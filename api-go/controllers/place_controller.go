package controllers

import (
	"fmt"
	"log"
	"math"
	"math/rand"
	"net/http"

	"github.com/snap-point/api-go/models"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"
)

type PlaceController struct {
	DB *gorm.DB
}

type NearbyPlacesQuery struct {
	Latitude  float64 `form:"latitude" binding:"required"`
	Longitude float64 `form:"longitude" binding:"required"`
	ZoomLevel int     `form:"zoomLevel" binding:"required,min=1,max=20"`
	Radius    float64 `form:"radius"`
}

type PlacePostsQuery struct {
	SortBy    string `form:"sortBy" binding:"omitempty,oneof=newest highest_rated most_liked"`
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"pageSize,default=10" binding:"min=1,max=50"`
	TimeFrame string `form:"timeFrame" binding:"omitempty,oneof=today this_week this_month all_time"`
}

func NewPlaceController(db *gorm.DB) *PlaceController {
	return &PlaceController{DB: db}
}

// GetNearbyPlaces godoc
// @Summary Get nearby places based on location and zoom level
// @Description Returns places near the specified location, filtered by zoom level and rating
// @Tags places
// @Accept json
// @Produce json
// @Param latitude query number true "User's latitude"
// @Param longitude query number true "User's longitude"
// @Param zoomLevel query integer true "Map zoom level (1-20)"
// @Param radius query number false "Search radius in kilometers"
// @Success 200 {array} models.Place
// @Router /places/nearby [get]
func (pc *PlaceController) GetNearbyPlaces(c *gin.Context) {
	var query NearbyPlacesQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// İstanbul'un merkezi koordinatları
	const (
		istanbulLat = 41.0082
		istanbulLon = 28.9784
	)

	// Arama yarıçapı (km cinsinden)
	radius := 20.0

	limit := 10 // İstenen yer sayısı

	var places []models.Place
	result := pc.DB.
		Select("*, (6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance", istanbulLat, istanbulLon, istanbulLat).
		Where("(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= ?", istanbulLat, istanbulLon, istanbulLat, radius).
		Order("distance").
		Limit(limit).
		Find(&places)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching places"})
		return
	}

	if len(places) < limit {
		artificialPlaces, err := pc.generateArtificialPlaces(istanbulLat, istanbulLon, limit-len(places))
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Error generating artificial places"})
			return
		}

		for _, place := range artificialPlaces {
			result := pc.DB.Create(&place)
			if result.Error != nil {
				log.Printf("Error saving artificial place: %v", result.Error)
				log.Printf("Place details: %+v", place)
			} else {
				places = append(places, place)
			}
		}
	}

	c.JSON(http.StatusOK, places)
}

func (pc *PlaceController) generateArtificialPlaces(centerLat, centerLon float64, count int) ([]models.Place, error) {
	const (
		minLat = 40.8
		maxLat = 41.2
		minLon = 28.5
		maxLon = 29.4
	)

	placeTypes := []string{"Park", "Restaurant", "Museum", "Cafe", "Shopping Center"}

	var artificialPlaces []models.Place

	for i := 0; i < count; i++ {
		lat := minLat + rand.Float64()*(maxLat-minLat)
		lon := minLon + rand.Float64()*(maxLon-minLon)

		place := models.Place{
			Name:        fmt.Sprintf("Artificial Place %d", i+1),
			Description: "This is an artificially generated place in Istanbul",
			Address:     "Generated Address in Istanbul",
			Latitude:    lat,
			Longitude:   lon,
			Rating:      float64(rand.Intn(5) + 1),
			PlaceType:   placeTypes[rand.Intn(len(placeTypes))],
			IsVerified:  false,
			IsGenerated: true,
			PriceLevel:  rand.Intn(4) + 1,
			OpeningTime: nil,
			ClosingTime: nil,
		}

		artificialPlaces = append(artificialPlaces, place)
	}

	return artificialPlaces, nil
}

// GetPlaceDetails godoc
// @Summary Get detailed information about a specific place
// @Description Returns detailed place information including recent posts and rating
// @Tags places
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Success 200 {object} models.Place
// @Router /places/{id} [get]
func (pc *PlaceController) GetPlaceDetails(c *gin.Context) {
	id := c.Param("id")

	var place models.Place
	result := pc.DB.
		Preload("Posts", func(db *gorm.DB) *gorm.DB {
			return db.Order("created_at DESC").Limit(10)
		}).
		First(&place, id)

	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	c.JSON(http.StatusOK, place)
}

// GetPlaceProfile godoc
// @Summary Get detailed profile information about a place
// @Description Returns comprehensive place information including stats and recent activity
// @Tags places
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Success 200 {object} map[string]interface{}
// @Router /places/{id}/profile [get]
func (pc *PlaceController) GetPlaceProfile(c *gin.Context) {
	id := c.Param("id")

	var place models.Place
	result := pc.DB.First(&place, id)
	if result.Error != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Get place statistics
	var stats struct {
		TotalPosts     int64   `json:"totalPosts"`
		TotalVisitors  int64   `json:"totalVisitors"`
		AverageRating  float64 `json:"averageRating"`
		TotalCheckIns  int64   `json:"totalCheckIns"`
		UniqueVisitors int64   `json:"uniqueVisitors"`
	}

	pc.DB.Model(&models.Post{}).Where("place_id = ?", id).Count(&stats.TotalPosts)
	pc.DB.Model(&models.ActivityLog{}).Where("place_id = ?", id).Count(&stats.TotalCheckIns)
	pc.DB.Model(&models.ActivityLog{}).Where("place_id = ?", id).Distinct("user_id").Count(&stats.UniqueVisitors)

	// Get recent activity
	var recentActivity []struct {
		models.ActivityLog
		Username string `json:"username"`
	}
	pc.DB.Model(&models.ActivityLog{}).
		Select("activity_logs.*, users.username").
		Joins("JOIN users ON users.id = activity_logs.user_id").
		Where("place_id = ?", id).
		Order("created_at DESC").
		Limit(5).
		Find(&recentActivity)

	// Get top contributors
	var topContributors []struct {
		UserID     uint   `json:"userId"`
		Username   string `json:"username"`
		PostCount  int64  `json:"postCount"`
		TotalLikes int64  `json:"totalLikes"`
	}
	pc.DB.Model(&models.Post{}).
		Select("posts.user_id, users.username, COUNT(posts.id) as post_count, SUM(COALESCE((SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id), 0)) as total_likes").
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.place_id = ?", id).
		Group("posts.user_id, users.username").
		Order("post_count DESC").
		Limit(10).
		Find(&topContributors)

	response := gin.H{
		"place":           place,
		"stats":           stats,
		"recentActivity":  recentActivity,
		"topContributors": topContributors,
	}

	c.JSON(http.StatusOK, response)
}

// GetPlacePosts godoc
// @Summary Get posts from a specific place with sorting and pagination
// @Description Returns paginated posts from a place with various sorting options
// @Tags places
// @Accept json
// @Produce json
// @Param id path string true "Place ID"
// @Param sortBy query string false "Sort posts by: newest, highest_rated, most_liked"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 10, max: 50)"
// @Param timeFrame query string false "Time frame: today, this_week, this_month, all_time"
// @Success 200 {object} map[string]interface{}
// @Router /places/{id}/posts [get]
func (pc *PlaceController) GetPlacePosts(c *gin.Context) {
	id := c.Param("id")
	var query PlacePostsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := pc.DB.Model(&models.Post{}).Where("place_id = ?", id)

	// Apply time frame filter
	switch query.TimeFrame {
	case "today":
		db = db.Where("created_at >= CURRENT_DATE")
	case "this_week":
		db = db.Where("created_at >= DATE_TRUNC('week', CURRENT_DATE)")
	case "this_month":
		db = db.Where("created_at >= DATE_TRUNC('month', CURRENT_DATE)")
	}

	// Apply sorting
	switch query.SortBy {
	case "highest_rated":
		db = db.Order("points DESC")
	case "most_liked":
		db = db.Order("(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) DESC")
	default: // "newest" or empty
		db = db.Order("created_at DESC")
	}

	// Calculate pagination
	offset := (query.Page - 1) * query.PageSize

	var total int64
	db.Count(&total)

	var posts []struct {
		models.Post
		LikesCount    int64  `json:"likesCount"`
		CommentsCount int64  `json:"commentsCount"`
		Username      string `json:"username"`
	}

	result := db.
		Select("posts.*, users.username, " +
			"(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count, " +
			"(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count").
		Joins("JOIN users ON users.id = posts.user_id").
		Offset(offset).
		Limit(query.PageSize).
		Find(&posts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching posts"})
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
