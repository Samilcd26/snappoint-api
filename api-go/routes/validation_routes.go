package routes

import (
	"github.com/gin-gonic/gin"
	"github.com/snap-point/api-go/controllers"
)

func SetupValidationRoutes(protected *gin.RouterGroup, validationController *controllers.ValidationController) {
	// User posts route
	users := protected.Group("/validation")
	{
		users.GET("/username/:username", validationController.ValidateUsername)
		users.GET("/email/:email", validationController.ValidateEmail)
	}
}
