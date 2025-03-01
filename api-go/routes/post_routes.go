package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupPostRoutes(protected *gin.RouterGroup, postController *controllers.PostController) {
	posts := protected.Group("/posts")
	{
		posts.POST("", postController.CreatePost)
		posts.PUT("/:id", postController.UpdatePost)
		posts.DELETE("/:id", postController.DeletePost)
	}

	// User posts route
	users := protected.Group("/users")
	{
		users.GET("/:id/posts", postController.GetUserPosts)
	}
}
