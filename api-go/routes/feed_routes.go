package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupFeedRoutes(protected *gin.RouterGroup, feedController *controllers.FeedController) {
	feed := protected.Group("/feed")
	{
		feed.GET("", feedController.GetUserFeed)
	}
}
