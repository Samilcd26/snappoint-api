package controllers

import (
	"context"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/credentials"
	"github.com/aws/aws-sdk-go-v2/service/s3"
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/snap-point/api-go/config"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
)

type UploadController struct {
	DB       *gorm.DB
	R2Client *s3.Client
	R2Config *config.R2Config
}

type PresignedURLRequest struct {
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
	MediaType   string `json:"mediaType" binding:"required,oneof=photo video"`
}

type AvatarUploadRequest struct {
	FileName    string `json:"fileName" binding:"required"`
	ContentType string `json:"contentType" binding:"required"`
	FileSize    int64  `json:"fileSize" binding:"required"`
}

type AvatarConfirmRequest struct {
	TempKey string `json:"tempKey" binding:"required"`
	UserID  uint   `json:"userId" binding:"required"`
}

type PresignedURLResponse struct {
	UploadURL    string `json:"uploadUrl"`
	FileURL      string `json:"fileUrl"`
	ThumbnailURL string `json:"thumbnailUrl,omitempty"`
	Key          string `json:"key"`
	ExpiresIn    int    `json:"expiresIn"`
}

type MultipleUploadRequest struct {
	Files []PresignedURLRequest `json:"files" binding:"required,dive"`
}

type MultipleUploadResponse struct {
	Files []PresignedURLResponse `json:"files"`
}

type UploadCompleteRequest struct {
	Key       string `json:"key" binding:"required"`
	MediaType string `json:"mediaType" binding:"required,oneof=photo video"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Duration  int    `json:"duration"`
}



func NewUploadController(db *gorm.DB) *UploadController {
	r2Config := config.GetR2Config()
	
	// Create R2 client
	r2Client := s3.New(s3.Options{
		BaseEndpoint: aws.String(fmt.Sprintf("https://%s.r2.cloudflarestorage.com", r2Config.AccountID)),
		Credentials: credentials.NewStaticCredentialsProvider(
			r2Config.AccessKeyID,
			r2Config.SecretAccessKey,
			"",
		),
		Region: r2Config.Region,
	})

	return &UploadController{
		DB:       db,
		R2Client: r2Client,
		R2Config: r2Config,
	}
}

func (uc *UploadController) GetPresignedURL(c *gin.Context) {
	user := utils.GetUser(c)
	var req PresignedURLRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate file type
	if !uc.isValidFileType(req.ContentType, req.MediaType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid file type for media type"})
		return
	}

	// Validate file size
	if !uc.isValidFileSize(req.FileSize, req.MediaType) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File size exceeds limit"})
		return
	}

	// Generate unique key
	key := uc.generateFileKey(user.UserID, req.FileName, req.MediaType)
	
	// Create presigned URL
	presignedURL, err := uc.createPresignedURL(key, req.ContentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload URL"})
		return
	}

	response := PresignedURLResponse{
		UploadURL: presignedURL,
		FileURL:   fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, key),
		Key:       key,
		ExpiresIn: 3600, // 1 hour
	}

	// Generate thumbnail URL for videos
	if req.MediaType == "video" {
		thumbnailKey := uc.generateThumbnailKey(key)
		response.ThumbnailURL = fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, thumbnailKey)
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    response,
		Message: "Presigned URL generated successfully",
	})
}

func (uc *UploadController) GetMultiplePresignedURLs(c *gin.Context) {
	user := utils.GetUser(c)
	var req MultipleUploadRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Validate number of files (max 10 for Instagram-like experience)
	if len(req.Files) > 10 {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Maximum 10 files allowed per upload"})
		return
	}

	var responses []PresignedURLResponse

	for _, fileReq := range req.Files {
		// Validate each file
		if !uc.isValidFileType(fileReq.ContentType, fileReq.MediaType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("Invalid file type for %s", fileReq.FileName),
			})
			return
		}

		if !uc.isValidFileSize(fileReq.FileSize, fileReq.MediaType) {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": fmt.Sprintf("File size exceeds limit for %s", fileReq.FileName),
			})
			return
		}

		// Generate unique key
		key := uc.generateFileKey(user.UserID, fileReq.FileName, fileReq.MediaType)
		
		// Create presigned URL
		presignedURL, err := uc.createPresignedURL(key, fileReq.ContentType)
		if err != nil {
			c.JSON(http.StatusInternalServerError, gin.H{
				"error": fmt.Sprintf("Failed to create upload URL for %s", fileReq.FileName),
			})
			return
		}

		response := PresignedURLResponse{
			UploadURL: presignedURL,
			FileURL:   fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, key),
			Key:       key,
			ExpiresIn: 3600,
		}

		if fileReq.MediaType == "video" {
			thumbnailKey := uc.generateThumbnailKey(key)
			response.ThumbnailURL = fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, thumbnailKey)
		}

		responses = append(responses, response)
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data: MultipleUploadResponse{
			Files: responses,
		},
		Message: "Multiple presigned URLs generated successfully",
	})
}

func (uc *UploadController) ConfirmUpload(c *gin.Context) {
	user := utils.GetUser(c)
	var req UploadCompleteRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	// Verify file exists in R2
	exists, err := uc.verifyFileExists(req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to verify file upload"})
		return
	}

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "File not found in storage"})
		return
	}

	// Get file info
	fileInfo, err := uc.getFileInfo(req.Key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to get file information"})
		return
	}

	response := gin.H{
		"key":       req.Key,
		"fileUrl":   fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, req.Key),
		"fileSize":  fileInfo.ContentLength,
		"mediaType": req.MediaType,
					"uploadedBy": user.UserID,
		"uploadedAt": time.Now(),
	}

	if req.MediaType == "video" {
		thumbnailKey := uc.generateThumbnailKey(req.Key)
		response["thumbnailUrl"] = fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, thumbnailKey)
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    response,
		Message: "Upload confirmed successfully",
	})
}

func (uc *UploadController) DeleteFile(c *gin.Context) {
	user := utils.GetUser(c)
	key := c.Param("key")
	
	if key == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "File key is required"})
		return
	}

	// Verify user owns this file (extract user ID from key)
	if !uc.verifyFileOwnership(key, user.UserID) {
		c.JSON(http.StatusForbidden, gin.H{"error": "Access denied"})
		return
	}

	// Delete from R2
	err := uc.deleteFile(key)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to delete file"})
		return
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Message: "File deleted successfully",
	})
}

func (uc *UploadController) GetAvatarTempURL(c *gin.Context) {
	var req AvatarUploadRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if !uc.isValidAvatarFile(req.ContentType, req.FileSize) {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid avatar file type or size"})
		return
	}

	key := uc.generateTempAvatarKey(req.FileName)
	
	presignedURL, err := uc.createPresignedURL(key, req.ContentType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to create upload URL"})
		return
	}

	response := PresignedURLResponse{
		UploadURL: presignedURL,
		FileURL:   fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, key),
		Key:       key,
		ExpiresIn: 1800,
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    response,
		Message: "Temporary avatar upload URL generated successfully",
	})
}

func (uc *UploadController) ConfirmAvatarUpload(c *gin.Context) {
	var req AvatarConfirmRequest
	
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	exists, err := uc.verifyFileExists(req.TempKey)
	if err != nil || !exists {
		c.JSON(http.StatusNotFound, gin.H{"error": "Temporary avatar file not found"})
		return
	}

	permanentKey := uc.generateAvatarKey(req.UserID, req.TempKey)
	
	err = uc.moveFile(req.TempKey, permanentKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to confirm avatar upload"})
		return
	}

	response := gin.H{
		"key":     permanentKey,
		"fileUrl": fmt.Sprintf("%s/%s", uc.R2Config.PublicURL, permanentKey),
		"userId":  req.UserID,
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Data:    response,
		Message: "Avatar upload confirmed successfully",
	})
}

func (uc *UploadController) CleanupTempAvatar(c *gin.Context) {
	tempKey := c.Param("tempKey")
	
	if tempKey == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Temp key is required"})
		return
	}

	if !strings.HasPrefix(tempKey, "temp/avatars/") {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid temp key format"})
		return
	}

	err := uc.deleteFile(tempKey)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Failed to cleanup temporary file"})
		return
	}

	c.JSON(http.StatusOK, StandardResponse{
		Success: true,
		Message: "Temporary avatar cleaned up successfully",
	})
}

// Helper functions
func (uc *UploadController) isValidFileType(contentType, mediaType string) bool {
	validTypes := map[string][]string{
		"photo": {
			"image/jpeg", "image/jpg", "image/png", "image/webp", "image/heic",
		},
		"video": {
			"video/mp4", "video/quicktime", "video/avi", "video/webm", "video/mov",
		},
	}

	allowed, exists := validTypes[mediaType]
	if !exists {
		return false
	}

	for _, validType := range allowed {
		if contentType == validType {
			return true
		}
	}
	return false
}

func (uc *UploadController) isValidFileSize(fileSize int64, mediaType string) bool {
	// Size limits in bytes
	limits := map[string]int64{
		"photo": 10 * 1024 * 1024,   // 10MB
		"video": 100 * 1024 * 1024,  // 100MB
	}

	limit, exists := limits[mediaType]
	if !exists {
		return false
	}

	return fileSize <= limit
}

func (uc *UploadController) generateFileKey(userID uint, fileName, mediaType string) string {
	ext := filepath.Ext(fileName)
	uuid := uuid.New().String()
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("uploads/%s/%d/%d_%s%s", mediaType, userID, timestamp, uuid, ext)
}

func (uc *UploadController) generateThumbnailKey(originalKey string) string {
	ext := filepath.Ext(originalKey)
	keyWithoutExt := strings.TrimSuffix(originalKey, ext)
	return fmt.Sprintf("%s_thumbnail.jpg", keyWithoutExt)
}

func (uc *UploadController) createPresignedURL(key, contentType string) (string, error) {
	input := &s3.PutObjectInput{
		Bucket:      aws.String(uc.R2Config.BucketName),
		Key:         aws.String(key),
		ContentType: aws.String(contentType),
	}

	presigner := s3.NewPresignClient(uc.R2Client)
	req, err := presigner.PresignPutObject(context.TODO(), input, func(opts *s3.PresignOptions) {
		opts.Expires = time.Hour // 1 hour expiry
	})

	if err != nil {
		return "", err
	}

	return req.URL, nil
}

func (uc *UploadController) verifyFileExists(key string) (bool, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(uc.R2Config.BucketName),
		Key:    aws.String(key),
	}

	_, err := uc.R2Client.HeadObject(context.TODO(), input)
	if err != nil {
		return false, nil
	}

	return true, nil
}

func (uc *UploadController) getFileInfo(key string) (*s3.HeadObjectOutput, error) {
	input := &s3.HeadObjectInput{
		Bucket: aws.String(uc.R2Config.BucketName),
		Key:    aws.String(key),
	}

	return uc.R2Client.HeadObject(context.TODO(), input)
}

func (uc *UploadController) deleteFile(key string) error {
	input := &s3.DeleteObjectInput{
		Bucket: aws.String(uc.R2Config.BucketName),
		Key:    aws.String(key),
	}

	_, err := uc.R2Client.DeleteObject(context.TODO(), input)
	return err
}

func (uc *UploadController) verifyFileOwnership(key string, userID uint) bool {
	// Extract user ID from key format: uploads/{mediaType}/{userID}/{timestamp}_{uuid}.{ext}
	parts := strings.Split(key, "/")
	if len(parts) < 3 {
		return false
	}

	keyUserID := parts[2]
	return fmt.Sprintf("%d", userID) == keyUserID
}

func (uc *UploadController) isValidAvatarFile(contentType string, fileSize int64) bool {
	validTypes := []string{
		"image/jpeg", "image/jpg", "image/png", "image/webp",
	}
	
	validType := false
	for _, validContentType := range validTypes {
		if contentType == validContentType {
			validType = true
			break
		}
	}
	
	if !validType {
		return false
	}
	
	// Avatar size limit: 5MB
	return fileSize <= 5*1024*1024
}

func (uc *UploadController) generateTempAvatarKey(fileName string) string {
	ext := filepath.Ext(fileName)
	uuid := uuid.New().String()
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("temp/avatars/%d_%s%s", timestamp, uuid, ext)
}

func (uc *UploadController) generateAvatarKey(userID uint, tempKey string) string {
	ext := filepath.Ext(tempKey)
	timestamp := time.Now().Unix()
	
	return fmt.Sprintf("users/%d/avatar/%d_avatar%s", userID, timestamp, ext)
}

func (uc *UploadController) moveFile(sourceKey, destKey string) error {
	copyInput := &s3.CopyObjectInput{
		Bucket:     aws.String(uc.R2Config.BucketName),
		CopySource: aws.String(fmt.Sprintf("%s/%s", uc.R2Config.BucketName, sourceKey)),
		Key:        aws.String(destKey),
	}
	
	_, err := uc.R2Client.CopyObject(context.TODO(), copyInput)
	if err != nil {
		return err
	}
	
	return uc.deleteFile(sourceKey)
} 