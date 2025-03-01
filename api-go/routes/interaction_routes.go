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

	// User interactions
	users := protected.Group("/users")
	{
		users.POST("/:id/follow", interactionController.FollowUser)
		users.GET("/:id/followers", interactionController.GetUserFollowers)
		users.GET("/:id/following", interactionController.GetUserFollowing)
	}
}
