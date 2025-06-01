package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupPostRoutes(protected *gin.RouterGroup, postController *controllers.PostController) {
	posts := protected.Group("/posts")
	{
		posts.POST("", postController.CreatePost)
		posts.GET("/:id", postController.GetPostDetail)
		posts.PUT("/:id", postController.UpdatePost)
		posts.DELETE("/:id", postController.DeletePost)
	}

	// User posts routes
	users := protected.Group("/users")
	{
		users.GET("/:userId/posts", postController.GetUserPosts)
		users.GET("/:userId/places/:placeId/posts", postController.GetUserPostsAtPlace)
	}

	// Place posts routes
	places := protected.Group("/places")
	{
		places.GET("/:placeId/posts/grid", postController.GetPlacePostsGrid)
	}
}
