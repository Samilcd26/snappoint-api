package config

import (
	"fmt"
	"log"
	"os"

	"github.com/joho/godotenv"
	"github.com/snap-point/api-go/models"
	"gorm.io/driver/postgres"
	"gorm.io/gorm"
)

type R2Config struct {
	AccountID       string
	AccessKeyID     string
	SecretAccessKey string
	BucketName      string
	PublicURL       string
	Region          string
}

func GetR2Config() *R2Config {
	return &R2Config{
		AccountID:       os.Getenv("CLOUDFLARE_ACCOUNT_ID"),
		AccessKeyID:     os.Getenv("CLOUDFLARE_ACCESS_KEY_ID"),
		SecretAccessKey: os.Getenv("CLOUDFLARE_SECRET_ACCESS_KEY"),
		BucketName:      os.Getenv("CLOUDFLARE_BUCKET_NAME"),
		PublicURL:       os.Getenv("CLOUDFLARE_PUBLIC_URL"),
		Region:          "auto",
	}
}

func ConnectDatabase() (*gorm.DB, error) {
	err := godotenv.Load()
	if err != nil {
		// Log the error but don't fail - might be in production without .env file
	}

	dsn := os.Getenv("DATABASE_URL")
	if dsn == "" {
		dsn = "host=localhost user=youruser dbname=yourdb port=5432 sslmode=disable TimeZone=Asia/Shanghai"
	}

	db, err := gorm.Open(postgres.Open(dsn), &gorm.Config{})
	if err != nil {
		return nil, err
	}

	return db, nil
}

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
