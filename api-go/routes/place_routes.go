package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupPlaceRoutes(protected *gin.RouterGroup, placeController *controllers.PlaceController) {
	places := protected.Group("/places")
	{
		places.GET("/nearby", placeController.GetNearbyPlaces)
		places.GET("/:id", placeController.GetPlaceDetails)
		places.GET("/:id/profile", placeController.GetPlaceProfile)
		places.GET("/:id/posts", placeController.GetPlacePosts)
	}
}
