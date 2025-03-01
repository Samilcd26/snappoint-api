package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
	"github.com/snap-point/api-go/middleware"
	"gorm.io/gorm"
)

func SetupRoutes(r *gin.Engine, db *gorm.DB) {
	// Initialize controllers
	authController := controllers.NewAuthController(db)
	postController := controllers.NewPostController(db)
	placeController := controllers.NewPlaceController(db)
	interactionController := controllers.NewInteractionController(db)
	feedController := controllers.NewFeedController(db)
	validationController := controllers.NewValidationController(db)

	// Public routes
	public := r.Group("/api")
	{
		public.POST("/register", authController.Register)
		public.POST("/login", authController.Login)
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

		// Setup other routes within the protected group
		SetupPostRoutes(protected, postController)
		SetupPlaceRoutes(protected, placeController)
		SetupInteractionRoutes(protected, interactionController)
		SetupFeedRoutes(protected, feedController)
		SetupValidationRoutes(protected, validationController)
	}
}
