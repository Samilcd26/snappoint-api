package controllers

import (
	"net/http"
	"os"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	DB *gorm.DB
}

func NewAuthController(db *gorm.DB) *AuthController {
	return &AuthController{DB: db}
}

func (ac *AuthController) Register(c *gin.Context) {
	var input struct {
		Username  string `json:"username" binding:"required"`
		Email     string `json:"email" binding:"required,email"`
		Password  string `json:"password" binding:"required,min=6"`
		FirstName string `json:"firstName"`
		LastName  string `json:"lastName"`
		Phone     string `json:"phone"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password"})
		return
	}

	user := models.User{
		Username:  input.Username,
		Email:     input.Email,
		Password:  string(hashedPassword),
		FirstName: input.FirstName,
		LastName:  input.LastName,
		Phone:     input.Phone,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists"})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User registered successfully", "user": user})
}

func (ac *AuthController) Login(c *gin.Context) {
	var input struct {
		Email    string `json:"email" binding:"required,email"`
		Password string `json:"password" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := ac.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Kullanıcının rollerini al
	var roles []string
	if err := ac.DB.Model(&user).Association("Roles").Find(&roles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user roles"})
		return
	}

	// Generate JWT token
	access_token_base := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"roles":   roles,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // Token expires in 7 days
	})

	refresh_token_base := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(), // Refresh token expires in 30 days
	})

	access_token, err := access_token_base.SignedString([]byte(os.Getenv("JWT_SECRET")))
	refresh_token, err := refresh_token_base.SignedString([]byte(os.Getenv("JWT_SECRET")))

	ac.DB.Create(&models.RefreshToken{
		UserID:         user.ID,
		Token:          refresh_token,
		ExpirationDate: time.Now().Add(time.Hour * 24 * 30), // Refresh token expires in 30 days
	})

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate token", "success": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"token_type":    "Bearer",
		"access_token":  access_token,
		"refresh_token": refresh_token,
		"user":          gin.H{"id": user.ID, "email": user.Email, "username": user.Username, "profilePicture": user.Avatar},
		"success":       true,
	})
}

func (ac *AuthController) RefreshToken(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	// Find the refresh token in the database
	var refreshToken models.RefreshToken
	if err := ac.DB.Where("token = ?", input.RefreshToken).First(&refreshToken).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid refresh token", "success": false})
		return
	}

	// Check if the refresh token is expired
	if time.Now().After(refreshToken.ExpirationDate) {
		// Delete the expired token
		ac.DB.Delete(&refreshToken)
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Refresh token expired", "success": false})
		return
	}

	// Get the user associated with the refresh token
	var user models.User
	if err := ac.DB.First(&user, refreshToken.UserID).Error; err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found", "success": false})
		return
	}

	// Get user roles
	var roles []string
	if err := ac.DB.Model(&user).Association("Roles").Find(&roles); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user roles", "success": false})
		return
	}

	// Generate new access token
	accessTokenBase := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"roles":   roles,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(), // Access token expires in 7 days
	})

	accessToken, err := accessTokenBase.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate access token", "success": false})
		return
	}

	// Generate new refresh token
	refreshTokenBase := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(), // Refresh token expires in 30 days
	})

	newRefreshToken, err := refreshTokenBase.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token", "success": false})
		return
	}

	// Update the existing refresh token in the database
	refreshToken.Token = newRefreshToken
	refreshToken.ExpirationDate = time.Now().Add(time.Hour * 24 * 30) // Refresh token expires in 30 days
	ac.DB.Save(&refreshToken)

	c.JSON(http.StatusOK, gin.H{
		"token_type":    "Bearer",
		"access_token":  accessToken,
		"refresh_token": newRefreshToken,
		"user":          gin.H{"id": user.ID, "email": user.Email, "username": user.Username, "profilePicture": user.Avatar},
		"success":       true,
	})
}

func (ac *AuthController) GetProfile(c *gin.Context) {
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	var dbUser models.User
	if err := ac.DB.First(&dbUser, user.UserID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"user": gin.H{
			"id":        dbUser.ID,
			"username":  dbUser.Username,
			"email":     dbUser.Email,
			"firstName": dbUser.FirstName,
			"lastName":  dbUser.LastName,
			"phone":     dbUser.Phone,
			"bio":       dbUser.Bio,
			"avatar":    dbUser.Avatar,
			"createdAt": dbUser.CreatedAt,
			"roles":     user.Roles,
		},
	})
}

func (ac *AuthController) UpdateProfile(c *gin.Context) {
	userID, _ := c.Get("user_id")

	var input struct {
		FullName string `json:"full_name"`
		Bio      string `json:"bio"`
		Avatar   string `json:"avatar"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	var user models.User
	if err := ac.DB.First(&user, userID).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "User not found"})
		return
	}

	updates := map[string]interface{}{
		"full_name": input.FullName,
		"bio":       input.Bio,
		"avatar":    input.Avatar,
	}

	if err := ac.DB.Model(&user).Updates(updates).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to update profile"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"message": "Profile updated successfully",
		"user": gin.H{
			"id":        user.ID,
			"username":  user.Username,
			"email":     user.Email,
			"firstName": user.FirstName,
			"lastName":  user.LastName,
			"phone":     user.Phone,
			"bio":       user.Bio,
			"avatar":    user.Avatar,
			"createdAt": user.CreatedAt,
		},
	})
}

func (ac *AuthController) Logout(c *gin.Context) {
	var input struct {
		RefreshToken string `json:"refresh_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	// Find and delete the refresh token from the database
	var refreshToken models.RefreshToken
	result := ac.DB.Where("token = ?", input.RefreshToken).Delete(&refreshToken)

	if result.RowsAffected == 0 {
		// Token not found, but we'll still return success
		c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully", "success": true})
		return
	}

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to logout", "success": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Logged out successfully", "success": true})
}
