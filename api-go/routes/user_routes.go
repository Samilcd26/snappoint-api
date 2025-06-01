package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupUserRoutes(protected *gin.RouterGroup, userController *controllers.UserController) {
	users := protected.Group("/users")
	{
		// User profile endpoints
		users.GET("/:userId/profile", userController.GetUserProfile)
		users.GET("/search", userController.SearchUsers)
		users.GET("/suggested", userController.GetSuggestedUsers)
		users.GET("/top", userController.GetTopUsers)
		users.GET("/nearby", userController.GetNearbyUsers)
		users.GET("/username/:username", userController.GetUsersByUsername)
		
		// User actions
		users.POST("/:userId/block", userController.BlockUser)
		users.POST("/:userId/report", userController.ReportUser)
		
		// User activity
		users.GET("/:userId/activity", userController.GetUserActivity)
	}
} 