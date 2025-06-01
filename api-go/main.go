package main

import (
	"log"
	"os"

	"github.com/gin-gonic/gin"
	"github.com/joho/godotenv"
	"github.com/snap-point/api-go/config"
	"github.com/snap-point/api-go/routes"
)

func main() {
	// Set up logging to stdout
	log.SetOutput(os.Stdout)
	log.SetFlags(log.LstdFlags | log.Lshortfile)

	if err := godotenv.Load(); err != nil {
		log.Fatal("Error loading .env file")
	}

	// Initialize database
	db := config.InitDB()

	// Create a new Gin router
	r := gin.Default()

	// Add logging middleware
	r.Use(gin.LoggerWithWriter(os.Stdout))

	// Initialize routes
	routes.SetupRoutes(r, db)

	// Start the server
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	log.Printf("Starting server on port %s", port)
	r.Run(":" + port)
}
