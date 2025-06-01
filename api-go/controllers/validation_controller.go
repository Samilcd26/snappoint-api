package controllers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/models"
	"gorm.io/gorm"
)

type ValidationController struct {
	DB *gorm.DB
}

func NewValidationController(db *gorm.DB) *ValidationController {
	return &ValidationController{DB: db}
}

func (vc *ValidationController) ValidateUsername(c *gin.Context) {
	username := c.Param("username")

	var user models.User
	result := vc.DB.Where("username = ?", username).First(&user)

	if result.Error == nil {
		// Username exists
		c.JSON(http.StatusOK, gin.H{"exists": true})
	} else if result.Error == gorm.ErrRecordNotFound {
		// Username doesn't exist
		c.JSON(http.StatusOK, gin.H{"exists": false})
	} else {
		// Database error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check username"})
	}
}

func (vc *ValidationController) ValidateEmail(c *gin.Context) {
	email := c.Param("email")

	var user models.User
	result := vc.DB.Where("email = ?", email).First(&user)

	if result.Error == nil {
		// Email exists
		c.JSON(http.StatusOK, gin.H{"exists": true})
	} else if result.Error == gorm.ErrRecordNotFound {
		// Email doesn't exist
		c.JSON(http.StatusOK, gin.H{"exists": false})
	} else {
		// Database error
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to check email"})
	}
}
