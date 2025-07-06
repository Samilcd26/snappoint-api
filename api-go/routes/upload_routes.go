package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupUploadRoutes(r *gin.RouterGroup, uploadController *controllers.UploadController) {
	upload := r.Group("/upload")
	{
		// Single file upload URL generation
		upload.POST("/presigned-url", uploadController.GetPresignedURL)
		
		// Multiple files upload URL generation (for carousel posts)
		upload.POST("/multiple-presigned-urls", uploadController.GetMultiplePresignedURLs)
		
		// Confirm upload completion
		upload.POST("/confirm", uploadController.ConfirmUpload)
		
		// Delete uploaded file
		upload.DELETE("/file/:key", uploadController.DeleteFile)
		
		// Avatar confirmation (protected route)
		upload.POST("/avatar/confirm", uploadController.ConfirmAvatarUpload)
	}
} 