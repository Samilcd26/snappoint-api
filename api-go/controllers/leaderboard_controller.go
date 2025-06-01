package controllers

import (
	"math"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type LeaderboardController struct {
	DB *gorm.DB
}

type LeaderboardQuery struct {
	TimeFilter  string  `form:"timeFilter" binding:"omitempty,oneof=all_time weekly monthly"`
	IsCategory  bool    `form:"isCategory"`
	IsNearby    bool    `form:"isNearby"`
	CategoryID  string  `form:"categoryId"`
	Page        int     `form:"page,default=1" binding:"min=1"`
	PageSize    int     `form:"pageSize,default=10" binding:"min=1,max=50"`
	Latitude    float64 `form:"latitude"`
	Longitude   float64 `form:"longitude"`
	MaxDistance float64 `form:"maxDistance,default=50"` // 50km default
}

func NewLeaderboardController(db *gorm.DB) *LeaderboardController {
	return &LeaderboardController{DB: db}
}

func (lc *LeaderboardController) GetLeaderboard(c *gin.Context) {
	var query LeaderboardQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Default to all_time if not specified
	if query.TimeFilter == "" {
		query.TimeFilter = "all_time"
	}

	// Get current user from context
	user := utils.GetUser(c)
	userID := user.UserID

	// Base query for users
	baseQuery := lc.DB.Model(&models.User{}).
		Where("is_verified = ?", true)

	// Build the query based on filters
	var selectClause, joinClause, whereClause string
	var queryParams []interface{}
	var orderByClause string // Rank ordering için kullanılacak alan

	// Start with basic user fields
	selectClause = "users.id, users.username, users.first_name, users.last_name, users.avatar"

	// Handle time filter
	switch query.TimeFilter {
	case "weekly":
		startOfWeek := time.Now().AddDate(0, 0, -int(time.Now().Weekday()))
		startOfWeek = time.Date(startOfWeek.Year(), startOfWeek.Month(), startOfWeek.Day(), 0, 0, 0, 0, time.Local)

		joinClause += " LEFT JOIN posts ON users.id = posts.user_id AND posts.created_at >= ?"
		queryParams = append(queryParams, startOfWeek)
		selectClause += ", COALESCE(SUM(posts.earned_points), 0) as points"
		orderByClause = "COALESCE(SUM(posts.earned_points), 0)" // Window function için

	case "monthly":
		startOfMonth := time.Date(time.Now().Year(), time.Now().Month(), 1, 0, 0, 0, 0, time.Local)

		joinClause += " LEFT JOIN posts ON users.id = posts.user_id AND posts.created_at >= ?"
		queryParams = append(queryParams, startOfMonth)
		selectClause += ", COALESCE(SUM(posts.earned_points), 0) as points"
		orderByClause = "COALESCE(SUM(posts.earned_points), 0)" // Window function için

	default: // all_time
		selectClause += ", users.total_points as points"
		orderByClause = "users.total_points" // Window function için
	}

	// Add category filter if specified
	if query.IsCategory {
		if query.CategoryID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Category ID is required when isCategory is true"})
			return
		}

		if query.TimeFilter == "all_time" {
			// For all_time, we need to join on posts and places
			joinClause = " LEFT JOIN posts ON users.id = posts.user_id"
		}

		joinClause += " LEFT JOIN places ON posts.place_id = places.id"
		whereClause += " AND ? = ANY(places.categories)"
		queryParams = append(queryParams, query.CategoryID)
	}

	// Add nearby filter if specified
	if query.IsNearby {
		if query.Latitude == 0 || query.Longitude == 0 {
			c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and longitude are required when isNearby is true"})
			return
		}

		// Limit max distance
		if query.MaxDistance > 100 {
			query.MaxDistance = 100 // Set an upper limit (100km)
		}

		// Add distance calculation to select
		distanceCalc := "(6371 * acos(cos(radians(?)) * cos(radians(posts.latitude)) * " +
			"cos(radians(posts.longitude) - radians(?)) + sin(radians(?)) * sin(radians(posts.latitude))))"

		selectClause += ", " + distanceCalc + " AS distance"
		queryParams = append(queryParams, query.Latitude, query.Longitude, query.Latitude)

		if joinClause == "" {
			joinClause = " LEFT JOIN posts ON users.id = posts.user_id"
		}

		whereClause += " AND " + distanceCalc + " <= ?"
		queryParams = append(queryParams, query.Latitude, query.Longitude, query.Latitude, query.MaxDistance)
	}

	// Complete the select clause with rank using direct field reference
	selectClause += ", RANK() OVER (ORDER BY " + orderByClause + " DESC) as rank"

	// Apply all clauses to the query
	baseQuery = baseQuery.Select(selectClause)

	// Apply joins if any
	if joinClause != "" {
		for _, param := range queryParams {
			joinClause = lc.DB.Statement.Dialector.Explain(joinClause, param)
		}
		baseQuery = baseQuery.Joins(joinClause)
	}

	// Apply where clause if any
	if whereClause != "" {
		baseQuery = baseQuery.Where(whereClause, queryParams...)
	}

	// Group by for aggregations
	baseQuery = baseQuery.Group("users.id, users.username, users.first_name, users.last_name, users.avatar")

	if query.TimeFilter == "all_time" {
		baseQuery = baseQuery.Group("users.total_points")
	}

	if query.IsNearby {
		baseQuery = baseQuery.Group("distance")
	}

	// Get total count for pagination
	var count int64
	countQuery := baseQuery.Session(&gorm.Session{})
	if err := countQuery.Count(&count).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error counting users: " + err.Error()})
		return
	}

	// Calculate pagination
	offset := (query.Page - 1) * query.PageSize

	// Kullanıcı sıralamasını temsil edecek struct
	type LeaderboardUser struct {
		ID        uint    `json:"id" gorm:"column:id"`
		Username  string  `json:"username" gorm:"column:username"`
		FirstName string  `json:"first_name" gorm:"column:first_name"`
		LastName  string  `json:"last_name" gorm:"column:last_name"`
		Avatar    string  `json:"avatar" gorm:"column:avatar"`
		Points    float64 `json:"points" gorm:"column:points"`
		Rank      int     `json:"rank" gorm:"column:rank"`
		Distance  float64 `json:"distance,omitempty" gorm:"column:distance"`
	}

	// Get top users for the current page
	var leaderboardUsers []LeaderboardUser
	if err := baseQuery.Order("rank").Offset(offset).Limit(query.PageSize).Scan(&leaderboardUsers).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching leaderboard: " + err.Error()})
		return
	}

	// Find the current user's rank (specific to the filter type)
	var userRank LeaderboardUser
	userRankQuery := baseQuery.Session(&gorm.Session{})
	err := userRankQuery.Where("users.id = ?", userID).Limit(1).Scan(&userRank).Error

	// Kullanıcı sıralamalarda yoksa
	if err != nil || userRank.ID == 0 {
		// Get the basic user info from the database
		var basicUserInfo struct {
			Username string `json:"username"`
		}
		lc.DB.Model(&models.User{}).Select("username").Where("id = ?", userID).First(&basicUserInfo)

		userRank = LeaderboardUser{
			ID:       userID,
			Rank:     0,
			Username: basicUserInfo.Username,
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"leaderboard": leaderboardUsers,
		"user_rank":   userRank,
		"pagination": gin.H{
			"current_page": query.Page,
			"page_size":    query.PageSize,
			"total_items":  count,
			"total_pages":  math.Ceil(float64(count) / float64(query.PageSize)),
		},
		"filter": gin.H{
			"time_filter":  query.TimeFilter,
			"is_category":  query.IsCategory,
			"category_id":  query.CategoryID,
			"is_nearby":    query.IsNearby,
			"max_distance": query.MaxDistance,
		},
	})
}
