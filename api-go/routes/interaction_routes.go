package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupInteractionRoutes(protected *gin.RouterGroup, interactionController *controllers.InteractionController) {
	// Post interactions
	posts := protected.Group("/posts")
	{
		posts.POST("/:id/like", interactionController.LikePost)
	}

	// User interactions - using :userId to be consistent with other routes
	users := protected.Group("/users")
	{
		users.POST("/:userId/follow", interactionController.FollowUser)
		users.GET("/:userId/followers", interactionController.GetUserFollowers)
		users.GET("/:userId/following", interactionController.GetUserFollowing)
	}
}
