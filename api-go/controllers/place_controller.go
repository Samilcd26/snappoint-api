package controllers

import (
	"math"
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/types"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type PlaceController struct {
	DB *gorm.DB
}

type NearbyPlacesQuery struct {
	Latitude       float64 `form:"latitude" binding:"required"`
	Longitude      float64 `form:"longitude" binding:"required"`
	ZoomLevel      int     `form:"zoomLevel" binding:"required,min=1,max=20"`
	Radius         float64 `form:"radius"`
	HideVisited    bool    `form:"hideVisited"`
	CategoryFilter string  `form:"category"`
	MaxPlaces      int     `form:"maxPlaces"`
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

type SimplifiedPlace struct {
	ID         uint           `json:"id"`
	Name       string         `json:"name"`
	Categories pq.StringArray `json:"categories"`
	Address    string         `json:"address"`
	Latitude   float64        `json:"latitude"`
	Longitude  float64        `json:"longitude"`
	BaseScore  int            `json:"baseScore"`
	PlaceType  string         `json:"placeType"`
	PlaceImage string         `json:"placeImage"`
	IsVerified bool           `json:"isVerified"`
	Features   pq.StringArray `json:"features"`
}

// GetNearbyPlaces godoc
// @Summary Get nearby places based on location and zoom level with filters
// @Tags places
// @Accept json
// @Produce json
// @Param latitude query number true "User's latitude"
// @Param longitude query number true "User's longitude"
// @Param zoomLevel query integer true "Map zoom level (1-20)"
// @Param radius query number false "Search radius in kilometers"
// @Param hideVisited query boolean false "Hide places already visited by the user"
// @Param userId query integer false "User ID (required if hideVisited is true)"
// @Param category query string false "Filter by category"
// @Param maxPlaces query integer false "Maximum number of places to return"
// @Success 200 {object} types.NearbyPlacesResponse
// @Router /places/nearby [get]
func (pc *PlaceController) GetNearbyPlaces(c *gin.Context) {
	// Get user from context
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	var query NearbyPlacesQuery
	
	// Try to bind query parameters first
	if err := c.ShouldBindQuery(&query); err != nil {
		// If direct binding fails, try to parse nested params format
		query.Latitude = parseFloat(c.Query("params[latitude]"))
		query.Longitude = parseFloat(c.Query("params[longitude]"))
		query.ZoomLevel = parseInt(c.Query("params[zoomLevel]"))
		query.Radius = parseFloat(c.Query("params[radius]"))
		query.HideVisited = parseBool(c.Query("params[hideVisited]"))
		query.CategoryFilter = c.Query("params[category]")
		query.MaxPlaces = parseInt(c.Query("params[maxPlaces]"))
		
		// Validate required fields
		if query.Latitude == 0 || query.Longitude == 0 || query.ZoomLevel == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "latitude, longitude, and zoomLevel are required",
				"debug": gin.H{
					"received_params": c.Request.URL.Query(),
					"parsed_values": gin.H{
						"latitude": query.Latitude,
						"longitude": query.Longitude,
						"zoomLevel": query.ZoomLevel,
					},
				},
			})
			return
		}
		
		// Validate zoom level range
		if query.ZoomLevel < 1 || query.ZoomLevel > 20 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "zoomLevel must be between 1 and 20",
			})
			return
		}
	}

	// Use user-provided coordinates and radius
	latitude := query.Latitude
	longitude := query.Longitude

	// Default radius to 20km if not specified
	radius := 20.0
	if query.Radius > 0 {
		radius = query.Radius
	}

	// Adjust radius based on zoom level (smaller radius for higher zoom)
	radius = radius * (21 - float64(query.ZoomLevel)) / 20

	// Limit number of places to return
	limit := 50 // Default number of places to return
	if query.MaxPlaces > 0 && query.MaxPlaces < limit {
		limit = query.MaxPlaces
	}

	// Build the query with conditional point_value based on user posts
	db := pc.DB.Model(&models.Place{}).
		Select(`id, latitude, longitude, 
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
				THEN 1 
				ELSE base_points 
			END as point_value, 
			is_verified, 
			(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance`,
			user.UserID, latitude, longitude, latitude).
		Where("(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= ?",
			latitude, longitude, latitude, radius)

	// Apply category filter if provided
	if query.CategoryFilter != "" {
		db = db.Where("? = ANY(categories)", query.CategoryFilter)
	}

	// Order by distance and limit results
	db = db.Order("distance").Limit(limit)

	var markers []types.Marker
	result := db.Find(&markers)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching places"})
		return
	}

	response := types.NearbyPlacesResponse{
		Markers: markers,
		Filters: struct {
			Radius      float64 `json:"radius"`
			ZoomLevel   int     `json:"zoomLevel"`
			HideVisited bool    `json:"hideVisited"`
			Category    string  `json:"category"`
		}{
			Radius:      radius,
			ZoomLevel:   query.ZoomLevel,
			HideVisited: query.HideVisited,
			Category:    query.CategoryFilter,
		},
	}

	c.JSON(http.StatusOK, response)
}

// GetPlaceProfile godoc
// @Summary Get detailed profile information about a place
// @Description Returns comprehensive place information including stats and recent activity
// @Tags places
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Success 200 {object} map[string]interface{}
// @Router /places/{placeId}/profile [get]
func (pc *PlaceController) GetPlaceProfile(c *gin.Context) {
	// Get user from context
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	id := c.Param("placeId")

	// Place temel bilgileri - kullanıcının post atıp atmadığına göre point_value hesapla
	var place struct {
		ID         uint    `json:"id"`
		Name       string  `json:"name"`
		Latitude   float64 `json:"latitude"`
		Longitude  float64 `json:"longitude"`
		PointValue int     `json:"point_value"`
		PlaceImage string  `json:"place_image"`
	}
	
	if err := pc.DB.Model(&models.Place{}).
		Select(`id, name, latitude, longitude, place_image,
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
				THEN 1 
				ELSE base_points 
			END as point_value`, user.UserID).
		Where("id = ?", id).
		First(&place).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Stat bilgileri
	var stats struct {
		TotalPosts    int64     `json:"totalPosts"`
		TotalPoints   int64     `json:"totalPoints"`
		UniquePosters int64     `json:"uniquePosters"`
		LastPostTime  time.Time `json:"lastPostTime"`
	}
	pc.DB.Model(&models.Post{}).Where("place_id = ?", id).Count(&stats.TotalPosts)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", id).Select("COALESCE(SUM(earned_points), 0)").Scan(&stats.TotalPoints)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", id).Distinct("user_id").Count(&stats.UniquePosters)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", id).Select("COALESCE(MAX(created_at), ?)", time.Time{}).Scan(&stats.LastPostTime)

	// Kullanıcıları grupla - her kullanıcının kaç post attığını göster
	var userPosts []struct {
		UserID      uint      `json:"userId"`
		Username    string    `json:"username"`
		FirstName   string    `json:"firstName"`
		LastName    string    `json:"lastName"`
		Avatar      string    `json:"avatar"`
		PostCount   int64     `json:"postCount"`
		TotalPoints int64     `json:"totalPoints"`
		LastPostAt  time.Time `json:"lastPostAt"`
	}

	pc.DB.Table("posts").
		Select(`
			users.id as user_id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			COUNT(posts.id) as post_count,
			COALESCE(SUM(posts.earned_points), 0) as total_points,
			MAX(posts.created_at) as last_post_at
		`).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.place_id = ?", id).
		Group("users.id, users.username, users.first_name, users.last_name, users.avatar").
		Order("post_count DESC, last_post_at DESC").
		Find(&userPosts)

	// En çok post atan ilk 5 kullanıcıyı getir (top users için ayrı)
	var topUsers []struct {
		UserID      uint   `json:"id"`
		Username    string `json:"username"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		TotalPoints int64  `json:"total_points"`
		PostCount   int64  `json:"post_count"`
		Avatar      string `json:"avatar"`
	}
	pc.DB.Table("posts").
		Select("user_id as user_id, users.username, users.first_name, users.last_name, users.total_points, users.avatar, COUNT(posts.id) as post_count").
		Joins("JOIN users ON users.id = posts.user_id").
		Where("place_id = ?", id).
		Group("user_id, users.username, users.first_name, users.last_name, users.total_points, users.avatar").
		Order("post_count DESC").
		Limit(5).
		Scan(&topUsers)

	response := gin.H{
		"id":          place.ID,
		"name":        place.Name,
		"latitude":    place.Latitude,
		"longitude":   place.Longitude,
		"point_value": place.PointValue,
		"place_image": place.PlaceImage,
		"stats":       stats,
		"user_posts":  userPosts,
		"top_users":   topUsers,
	}

	c.JSON(http.StatusOK, response)
}

// GetPlacePosts godoc
// @Summary Get posts from a specific place with sorting and pagination
// @Description Returns paginated posts from a place with various sorting options
// @Tags places
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Param sortBy query string false "Sort posts by: newest, highest_rated, most_liked"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 10, max: 50)"
// @Param timeFrame query string false "Time frame: today, this_week, this_month, all_time"
// @Success 200 {object} map[string]interface{}
// @Router /places/{placeId}/posts [get]
func (pc *PlaceController) GetPlacePosts(c *gin.Context) {
	id := c.Param("placeId")
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

// Helper functions for parsing query parameters
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

func parseBool(s string) bool {
	if s == "" {
		return false
	}
	val, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return val
}
