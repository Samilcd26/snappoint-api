package controllers

import (
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type UserController struct {
	DB *gorm.DB
}

func NewUserController(db *gorm.DB) *UserController {
	return &UserController{DB: db}
}

func (uc *UserController) GetUserProfile(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	userID := c.Param("userId")
	
	var targetUser models.User
	if err := uc.DB.Preload("Following").Preload("Followers").First(&targetUser, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var stats struct {
		PostsCount     int64 `json:"postsCount"`
		FollowersCount int64 `json:"followersCount"`
		FollowingCount int64 `json:"followingCount"`
	}

	uc.DB.Model(&models.Post{}).Where("user_id = ?", userID).Count(&stats.PostsCount)
	uc.DB.Model(&models.Follow{}).Where("following_user_id = ? AND status = ?", userID, "accepted").Count(&stats.FollowersCount)
	uc.DB.Model(&models.Follow{}).Where("follower_user_id = ? AND status = ?", userID, "accepted").Count(&stats.FollowingCount)

	var isFollowing bool
	var isFollowRequestPending bool
	if currentUser.UserID != targetUser.ID {
		var follow models.Follow
		result := uc.DB.Where("follower_user_id = ? AND following_user_id = ?", currentUser.UserID, userID).First(&follow)
		if result.Error == nil {
			isFollowing = follow.Status == "accepted"
			isFollowRequestPending = follow.Status == "pending"
		}
	}

	isOwnProfile := currentUser.UserID == targetUser.ID

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"data": gin.H{
			"id":               targetUser.ID,
			"username":         targetUser.Username,
			"firstName":        targetUser.FirstName,
			"lastName":         targetUser.LastName,
			"email":            targetUser.Email,
			"phone":            targetUser.Phone,
			"bio":              targetUser.Bio,
			"avatar":           targetUser.Avatar,
			"gender":           targetUser.Gender,
			"birthday":         targetUser.Birthday,
			"totalPoints":      targetUser.TotalPoints,
			"accountStatus":    targetUser.AccountStatus,
			"isVerified":       targetUser.IsVerified,
			"emailVerified":    targetUser.EmailVerified,
			"phoneVerified":    targetUser.PhoneVerified,
			"createdAt":        targetUser.CreatedAt,
			"isOwnProfile":     isOwnProfile,
			"isFollowing":      isFollowing,
			"isFollowPending":  isFollowRequestPending,
			"postsCount":       stats.PostsCount,
			"followersCount":   stats.FollowersCount,
			"followingCount":   stats.FollowingCount,
		},
	})
}

func (uc *UserController) SearchUsers(c *gin.Context) {
	query := strings.TrimSpace(c.Query("q"))
	if query == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Search query is required"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	offset := (page - 1) * pageSize

	var users []struct {
		ID           uint   `json:"id"`
		Username     string `json:"username"`
		FirstName    string `json:"firstName"`
		LastName     string `json:"lastName"`
		Avatar       string `json:"avatar"`
		IsVerified   bool   `json:"isVerified"`
		TotalPoints  int64  `json:"totalPoints"`
		PostsCount   int64  `json:"postsCount"`
	}

	searchPattern := "%" + query + "%"
	
	uc.DB.Table("users").
		Select(`
			users.id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			users.is_verified,
			users.total_points,
			COUNT(posts.id) as posts_count
		`).
		Joins("LEFT JOIN posts ON posts.user_id = users.id").
		Where("users.username ILIKE ? OR users.first_name ILIKE ? OR users.last_name ILIKE ?", 
			searchPattern, searchPattern, searchPattern).
		Group("users.id").
		Order("users.total_points DESC, posts_count DESC").
		Offset(offset).
		Limit(pageSize).
		Scan(&users)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"users":   users,
		"query":   query,
		"page":    page,
		"pageSize": pageSize,
	})
}

func (uc *UserController) GetSuggestedUsers(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "10"))

	var suggestedUsers []struct {
		ID          uint   `json:"id"`
		Username    string `json:"username"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Avatar      string `json:"avatar"`
		IsVerified  bool   `json:"isVerified"`
		TotalPoints int64  `json:"totalPoints"`
		Reason      string `json:"reason"`
	}

	uc.DB.Table("users").
		Select(`
			users.id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			users.is_verified,
			users.total_points,
			'popular' as reason
		`).
		Where(`
			users.id != ? AND 
			users.id NOT IN (
				SELECT following_user_id FROM follows 
				WHERE follower_user_id = ?
			)
		`, currentUser.UserID, currentUser.UserID).
		Order("users.total_points DESC").
		Limit(limit).
		Scan(&suggestedUsers)

	c.JSON(http.StatusOK, gin.H{
		"success":        true,
		"suggestedUsers": suggestedUsers,
	})
}

func (uc *UserController) GetUsersByUsername(c *gin.Context) {
	username := c.Param("username")
	
	var users []struct {
		ID          uint   `json:"id"`
		Username    string `json:"username"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Avatar      string `json:"avatar"`
		IsVerified  bool   `json:"isVerified"`
		TotalPoints int64  `json:"totalPoints"`
	}

	uc.DB.Table("users").
		Select("id, username, first_name, last_name, avatar, is_verified, total_points").
		Where("username ILIKE ?", "%"+username+"%").
		Order("total_points DESC").
		Limit(20).
		Scan(&users)

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"users":   users,
	})
}

func (uc *UserController) GetNearbyUsers(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	lat, _ := strconv.ParseFloat(c.Query("lat"), 64)
	lng, _ := strconv.ParseFloat(c.Query("lng"), 64)
	radius, _ := strconv.ParseFloat(c.DefaultQuery("radius", "10"), 64)

	if lat == 0 || lng == 0 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Latitude and longitude are required"})
		return
	}

	var nearbyUsers []struct {
		ID          uint    `json:"id"`
		Username    string  `json:"username"`
		FirstName   string  `json:"firstName"`
		LastName    string  `json:"lastName"`
		Avatar      string  `json:"avatar"`
		IsVerified  bool    `json:"isVerified"`
		TotalPoints int64   `json:"totalPoints"`
		Distance    float64 `json:"distance"`
		LastSeen    string  `json:"lastSeen"`
	}

	uc.DB.Table("users").
		Select(`
			DISTINCT users.id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			users.is_verified,
			users.total_points,
			ROUND(
				6371 * acos(
					cos(radians(?)) * 
					cos(radians(posts.latitude)) * 
					cos(radians(posts.longitude) - radians(?)) + 
					sin(radians(?)) * 
					sin(radians(posts.latitude))
				)::numeric, 2
			) AS distance,
			MAX(posts.created_at)::text as last_seen
		`, lat, lng, lat).
		Joins("JOIN posts ON posts.user_id = users.id").
		Where(`
			users.id != ? AND
			6371 * acos(
				cos(radians(?)) * 
				cos(radians(posts.latitude)) * 
				cos(radians(posts.longitude) - radians(?)) + 
				sin(radians(?)) * 
				sin(radians(posts.latitude))
			) <= ?
		`, currentUser.UserID, lat, lng, lat, radius).
		Group("users.id").
		Order("distance ASC, users.total_points DESC").
		Limit(50).
		Scan(&nearbyUsers)

	c.JSON(http.StatusOK, gin.H{
		"success":     true,
		"nearbyUsers": nearbyUsers,
		"radius":      radius,
		"center": gin.H{
			"lat": lat,
			"lng": lng,
		},
	})
}

func (uc *UserController) GetTopUsers(c *gin.Context) {
	timeFilter := c.DefaultQuery("timeFilter", "all_time")
	limit, _ := strconv.Atoi(c.DefaultQuery("limit", "50"))

	var topUsers []struct {
		ID          uint   `json:"id"`
		Username    string `json:"username"`
		FirstName   string `json:"firstName"`
		LastName    string `json:"lastName"`
		Avatar      string `json:"avatar"`
		IsVerified  bool   `json:"isVerified"`
		TotalPoints int64  `json:"totalPoints"`
		Rank        int    `json:"rank"`
	}

	query := uc.DB.Table("users").
		Select(`
			id,
			username,
			first_name,
			last_name,
			avatar,
			is_verified,
			total_points,
			ROW_NUMBER() OVER (ORDER BY total_points DESC) as rank
		`)

	switch timeFilter {
	case "today":
		query = query.Where("DATE(updated_at) = CURRENT_DATE")
	case "week":
		query = query.Where("updated_at >= CURRENT_DATE - INTERVAL '7 days'")
	case "month":
		query = query.Where("updated_at >= CURRENT_DATE - INTERVAL '30 days'")
	}

	query.Order("total_points DESC").
		Limit(limit).
		Scan(&topUsers)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"topUsers":   topUsers,
		"timeFilter": timeFilter,
	})
}

func (uc *UserController) BlockUser(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	targetUserID := c.Param("userId")
	
	if strconv.Itoa(int(currentUser.UserID)) == targetUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot block yourself"})
		return
	}

	var targetUser models.User
	if err := uc.DB.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	var existingBlock models.Block
	result := uc.DB.Where("blocker_user_id = ? AND blocked_user_id = ?", currentUser.UserID, targetUserID).First(&existingBlock)

	if result.Error == gorm.ErrRecordNotFound {
		block := models.Block{
			BlockerUserID: currentUser.UserID,
			BlockedUserID: targetUser.ID,
		}

		if err := uc.DB.Create(&block).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to block user"})
			return
		}

		uc.DB.Where("(follower_user_id = ? AND following_user_id = ?) OR (follower_user_id = ? AND following_user_id = ?)",
			currentUser.UserID, targetUserID, targetUserID, currentUser.UserID).Delete(&models.Follow{})

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User blocked successfully",
			"blocked": true,
		})
	} else {
		if err := uc.DB.Delete(&existingBlock).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to unblock user"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "User unblocked successfully",
			"blocked": false,
		})
	}
}

func (uc *UserController) ReportUser(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	targetUserID := c.Param("userId")
	
	var input struct {
		Reason      string `json:"reason" binding:"required"`
		Description string `json:"description"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if strconv.Itoa(int(currentUser.UserID)) == targetUserID {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Cannot report yourself"})
		return
	}

	var targetUser models.User
	if err := uc.DB.First(&targetUser, targetUserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	report := models.Report{
		ReporterUserID: currentUser.UserID,
		ReportedUserID: targetUser.ID,
		Reason:         input.Reason,
		Description:    input.Description,
		Status:         "pending",
	}

	if err := uc.DB.Create(&report).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to submit report"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Report submitted successfully",
	})
}

func (uc *UserController) GetUserActivity(c *gin.Context) {
	currentUser := utils.GetUser(c)
	if currentUser == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	userID := c.Param("userId")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("pageSize", "20"))
	offset := (page - 1) * pageSize

	if strconv.Itoa(int(currentUser.UserID)) != userID {
		c.JSON(http.StatusForbidden, gin.H{"error": "Can only view own activity"})
		return
	}

	var activities []models.ActivityLog
	uc.DB.Where("user_id = ?", userID).
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&activities)

	c.JSON(http.StatusOK, gin.H{
		"success":    true,
		"activities": activities,
		"page":       page,
		"pageSize":   pageSize,
	})
} 