package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
	"github.com/snap-point/api-go/middleware"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Initialize controllers
	uploadController := controllers.NewUploadController(db)
	authController := controllers.NewAuthController(db, uploadController)
	userController := controllers.NewUserController(db)
	postController := controllers.NewPostController(db)
	placeController := controllers.NewPlaceController(db)
	interactionController := controllers.NewInteractionController(db)
	feedController := controllers.NewFeedController(db)
	validationController := controllers.NewValidationController(db)
	leaderboardController := controllers.NewLeaderboardController(db)

	// Public routes
	public := r.Group("/api")
	{
		public.POST("/register", authController.Register)
		public.POST("/register/check-email", authController.RegisterEmailCheck)
		public.POST("/register/check-username", authController.RegisterUsernameCheck)
		public.POST("/verify-email", authController.VerifyEmail)
		public.POST("/login", authController.Login)
		public.POST("/google-login", authController.GoogleLogin)
	}

	// Public upload routes (no auth required for avatar during registration)
	publicUpload := r.Group("/api")
	{
		publicUpload.POST("/upload/avatar/temp", uploadController.GetAvatarTempURL)
		publicUpload.DELETE("/upload/avatar/temp/:tempKey", uploadController.CleanupTempAvatar)
	}

	// Protected routes
	protected := r.Group("/api")
	protected.Use(middleware.AuthMiddleware())
	{
		protected.POST("/logout", authController.Logout)
		protected.POST("/refresh-token", authController.RefreshToken)
		// User routes
		protected.GET("/profile", authController.GetProfile)
		protected.PUT("/profile", authController.UpdateProfile)

		//Leaderboard routes
		protected.GET("/leaderboard", leaderboardController.GetLeaderboard)

		// Setup other routes within the protected group
		SetupUserRoutes(protected, userController)
		SetupPostRoutes(protected, postController)
		SetupPlaceRoutes(protected, placeController)
		SetupInteractionRoutes(protected, interactionController)
		SetupFeedRoutes(protected, feedController)
		SetupValidationRoutes(protected, validationController)
		SetupUploadRoutes(protected, uploadController)
	}
}
