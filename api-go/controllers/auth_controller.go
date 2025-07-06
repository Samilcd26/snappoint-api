package controllers

import (
	"fmt"
	"net/http"
	"os"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/dgrijalva/jwt-go"
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/config"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/utils"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthController struct {
	DB               *gorm.DB
	GoogleConfig     *config.GoogleConfig
	UploadController *UploadController
}

// validateUsernamePattern validates username format and constraints
func validateUsernamePattern(username string) error {
	// Remove spaces for validation but keep original case
	trimmedUsername := strings.TrimSpace(username)
	
	// Check minimum length
	if len(trimmedUsername) < 3 {
		return fmt.Errorf("username must be at least 3 characters long")
	}
	
	// Check maximum length
	if len(trimmedUsername) > 20 {
		return fmt.Errorf("username must be no more than 20 characters long")
	}
	
	// Check if username starts with a letter
	startsWithLetter, _ := regexp.MatchString(`^[a-zA-Z]`, trimmedUsername)
	if !startsWithLetter {
		return fmt.Errorf("username must start with a letter")
	}
	
	// Check if username contains only allowed characters (letters, numbers, underscore)
	validPattern, _ := regexp.MatchString(`^[a-zA-Z][a-zA-Z0-9_]*$`, trimmedUsername)
	if !validPattern {
		return fmt.Errorf("username can only contain letters, numbers, and underscores")
	}
	
	// Check for reserved usernames
	reserved := []string{"admin", "root", "api", "www", "mail", "ftp", "test", "demo", "user", "guest", "null", "undefined"}
	for _, reservedWord := range reserved {
		if strings.ToLower(trimmedUsername) == reservedWord {
			return fmt.Errorf("this username is reserved and cannot be used")
		}
	}
	
	return nil
}

func NewAuthController(db *gorm.DB, uploadController *UploadController) *AuthController {
	return &AuthController{
		DB:               db,
		GoogleConfig:     config.NewGoogleConfig(),
		UploadController: uploadController,
	}
}

func (ac *AuthController) Register(c *gin.Context) {
	var input struct {
		Username     string `json:"username" binding:"required"`
		Email        string `json:"email" binding:"required,email"`
		Password     string `json:"password" binding:"required,min=6"`
		FirstName    string `json:"firstName" binding:"required"`
		LastName     string `json:"lastName" binding:"required"`
		Gender       string `json:"gender"`
		Birthday     string `json:"birthday"`
		Phone        string `json:"phone"`
		Avatar       string `json:"avatar"`
		AvatarTempKey string `json:"avatarTempKey"`
	}

	
	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}
	
	// Validate username pattern
	if err := validateUsernamePattern(input.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}
	
	// Hash password
	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(input.Password), bcrypt.DefaultCost)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not hash password", "success": false})
		return
	}

	hashedPasswordStr := string(hashedPassword)
	
	// Parse birthday if provided
	var birthday *time.Time
	if input.Birthday != "" {
		if parsed, err := time.Parse("2006-01-02", input.Birthday); err == nil {
			birthday = &parsed
		}
	}
	
	// Handle phone field - use nil if empty
	var phone *string
	if input.Phone != "" {
		phone = &input.Phone
	}
	
	user := models.User{
		Username:    input.Username,
		Email:       input.Email,
		Password:    &hashedPasswordStr,
		FirstName:   input.FirstName,
		LastName:    input.LastName,
		Gender:      input.Gender,
		Birthday:    birthday,
		Phone:       phone,
		Avatar:      input.Avatar,
		GoogleID:    nil, // Explicitly set to nil for email registration
		RoleID:      1, // Default role
		Provider:    "email",
		TotalPoints: 0,
		IsVerified:  false,
		EmailVerified: false,
		PhoneVerified: false,
	}

	if err := ac.DB.Create(&user).Error; err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Username or email already exists", "success": false})
		return
	}

	var finalAvatarURL string
	if input.AvatarTempKey != "" {
		finalAvatarURL = ac.confirmAvatarUpload(input.AvatarTempKey, user.ID)
		if finalAvatarURL != "" {
			user.Avatar = finalAvatarURL
			ac.DB.Save(&user)
		}
	}

	

	response := gin.H{
		"success": true,
		"message": "User registered successfully", 
		"user": gin.H{
			"id": user.ID,
			"email": user.Email,
			"username": user.Username,
			"firstName": user.FirstName,
			"lastName": user.LastName,
		},
	}

	if finalAvatarURL != "" {
		response["user"].(gin.H)["avatar"] = finalAvatarURL
	}

	c.JSON(http.StatusCreated, response)
}

func (ac *AuthController) VerifyEmail(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	var user models.User
	if err := ac.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Email not found", "success": false})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"success": true,
		"message": "Email verified successfully",
		"user_id": user.ID,
	})
}

func (ac *AuthController) RegisterEmailCheck(c *gin.Context) {
	var input struct {
		Email string `json:"email" binding:"required,email"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	var user models.User
	if err := ac.DB.Where("email = ?", input.Email).First(&user).Error; err != nil {
		// Email not found - good for registration
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Email available for registration",
			"available": true,
		})
		return
	}

	// Email already exists
	c.JSON(http.StatusConflict, gin.H{
		"success": false,
		"error": "Email already registered",
		"available": false,
	})
}

func (ac *AuthController) RegisterUsernameCheck(c *gin.Context) {
	var input struct {
		Username string `json:"username" binding:"required"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	// Validate username pattern
	if err := validateUsernamePattern(input.Username); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"error": err.Error(),
			"available": false,
		})
		return
	}

	var user models.User
	if err := ac.DB.Where("username = ?", input.Username).First(&user).Error; err != nil {
		// Username not found - good for registration
		c.JSON(http.StatusOK, gin.H{
			"success": true,
			"message": "Username available for registration",
			"available": true,
		})
		return
	}

	// Username already exists
	c.JSON(http.StatusConflict, gin.H{
		"success": false,
		"error": "Username already taken",
		"available": false,
	})
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

	if user.Password == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	if err := bcrypt.CompareHashAndPassword([]byte(*user.Password), []byte(input.Password)); err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid credentials"})
		return
	}

	// Get user role
	var role models.Role
	if err := ac.DB.First(&role, user.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user role"})
		return
	}

	// Generate JWT token
	access_token_base := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    role.Name,
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

	// Get user role
	var role models.Role
	if err := ac.DB.First(&role, user.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user role", "success": false})
		return
	}

	// Generate new access token
	accessTokenBase := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    role.Name,
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
		"success": true,
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
			"role":      user.Role,
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

func (ac *AuthController) GoogleLogin(c *gin.Context) {
	var input struct {
		IDToken      string `json:"id_token"`
		AccessToken  string `json:"access_token"`
		Code         string `json:"code"`
		RedirectURI  string `json:"redirect_uri"`
	}

	if err := c.ShouldBindJSON(&input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error(), "success": false})
		return
	}

	// Verify Google ID token or exchange code
	var userInfo *config.GoogleUserInfo
	var err error

	if input.Code != "" && input.RedirectURI != "" {
		// Exchange authorization code for tokens
		ctx := c.Request.Context()
		token, err := ac.GoogleConfig.ExchangeCode(ctx, input.Code)
		if err != nil {
			c.JSON(http.StatusUnauthorized, gin.H{"error": "Failed to exchange code for token", "success": false})
			return
		}
		
		userInfo, err = ac.GoogleConfig.GetUserInfo(token.AccessToken)
	} else if input.IDToken != "" {
		userInfo, err = ac.GoogleConfig.VerifyIDToken(input.IDToken)
	} else if input.AccessToken != "" {
		userInfo, err = ac.GoogleConfig.GetUserInfo(input.AccessToken)
	} else {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Either code with redirect_uri, id_token, or access_token is required", "success": false})
		return
	}

	if err != nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "Invalid Google token", "success": false})
		return
	}

	// Check if user already exists
	var user models.User
	userExists := ac.DB.Where("google_id = ? OR email = ?", userInfo.ID, userInfo.Email).First(&user).Error == nil

	if userExists {
		// Update existing user's Google info if needed
		if user.GoogleID == nil || *user.GoogleID == "" {
			user.GoogleID = &userInfo.ID
			user.Provider = "google"
			user.ProviderID = userInfo.ID
			if user.Avatar == "" && userInfo.Picture != "" {
				user.Avatar = userInfo.Picture
			}
			ac.DB.Save(&user)
		}
	} else {
		// Create new user
		// Generate unique username from email
		username := userInfo.Email
		counter := 1
		for {
			var existingUser models.User
			if ac.DB.Where("username = ?", username).First(&existingUser).Error != nil {
				break
			}
			username = userInfo.Email + strconv.Itoa(counter)
			counter++
		}

		// Get default role (assume role ID 1 is user role)
		var defaultRole models.Role
		if err := ac.DB.Where("name = ?", "user").First(&defaultRole).Error; err != nil {
			// If no user role found, use role ID 1 as fallback
			defaultRole.ID = 1
		}

		user = models.User{
			Username:      username,
			Email:         userInfo.Email,
			FirstName:     userInfo.GivenName,
			LastName:      userInfo.FamilyName,
			Avatar:        userInfo.Picture,
			GoogleID:      &userInfo.ID,
			Provider:      "google",
			ProviderID:    userInfo.ID,
			RoleID:        defaultRole.ID,
			EmailVerified: userInfo.VerifiedEmail,
			IsVerified:    userInfo.VerifiedEmail,
		}

		if err := ac.DB.Create(&user).Error; err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create user", "success": false})
			return
		}
	}

	// Get user role
	var role models.Role
	if err := ac.DB.First(&role, user.RoleID).Error; err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not fetch user role", "success": false})
		return
	}

	// Generate JWT tokens
	accessTokenBase := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"role":    role.Name,
		"exp":     time.Now().Add(time.Hour * 24 * 7).Unix(),
	})

	refreshTokenBase := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
		"user_id": user.ID,
		"exp":     time.Now().Add(time.Hour * 24 * 30).Unix(),
	})

	accessToken, err := accessTokenBase.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate access token", "success": false})
		return
	}

	refreshToken, err := refreshTokenBase.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Could not generate refresh token", "success": false})
		return
	}

	// Save refresh token
	ac.DB.Create(&models.RefreshToken{
		UserID:         user.ID,
		Token:          refreshToken,
		ExpirationDate: time.Now().Add(time.Hour * 24 * 30),
	})

	c.JSON(http.StatusOK, gin.H{
		"token_type":    "Bearer",
		"access_token":  accessToken,
		"refresh_token": refreshToken,
		"user":          gin.H{"id": user.ID, "email": user.Email, "username": user.Username, "profilePicture": user.Avatar},
		"success":       true,
	})
}

func (ac *AuthController) confirmAvatarUpload(tempKey string, userID uint) string {
	if ac.UploadController == nil {
		return ""
	}

	permanentKey := ac.UploadController.generateAvatarKey(userID, tempKey)
	
	err := ac.UploadController.moveFile(tempKey, permanentKey)
	if err != nil {
		return ""
	}

	return fmt.Sprintf("%s/%s", ac.UploadController.R2Config.PublicURL, permanentKey)
}
