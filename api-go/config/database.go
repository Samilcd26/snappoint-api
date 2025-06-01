package config

import (
	"fmt"
	"log"
	"os"

	"github.com/snap-point/api-go/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

func InitDB() *gorm.DB {
	dbHost := os.Getenv("DB_HOST")
	dbUser := os.Getenv("DB_USER")
	dbPassword := os.Getenv("DB_PASSWORD")
	dbName := os.Getenv("DB_NAME")
	dbPort := os.Getenv("DB_PORT")

	dsn := fmt.Sprintf("host=%s user=%s password=%s dbname=%s port=%s sslmode=disable",
		dbHost, dbUser, dbPassword, dbName, dbPort)

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		log.Fatal("Failed to connect to database:", err)
	}

	// Auto Migrate models
	db.AutoMigrate(&models.User{}, &models.RefreshToken{}, &models.Post{}, &models.Comment{}, &models.Like{}, &models.Follow{}, &models.Place{}, &models.ActivityLog{}, &models.Role{}, &models.PostMedia{})

	return db
}
