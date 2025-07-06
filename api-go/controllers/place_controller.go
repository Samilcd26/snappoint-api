package controllers

import (
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/lib/pq"
	"github.com/snap-point/api-go/models"
	"github.com/snap-point/api-go/types"
	"github.com/snap-point/api-go/utils"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type PlaceController struct {
	DB *gorm.DB
}

type NearbyPlacesQuery struct {
	Latitude       float64 `form:"latitude" binding:"required"`
	Longitude      float64 `form:"longitude" binding:"required"`
	ZoomLevel      int     `form:"zoomLevel" binding:"required,min=1,max=20"`
	Radius         float64 `form:"radius"`
	HideVisited    bool    `form:"hideVisited"`
	CategoryFilter string  `form:"category"`
	MaxPlaces      int     `form:"maxPlaces"`
}

type PlacePostsQuery struct {
	SortBy    string `form:"sortBy" binding:"omitempty,oneof=newest highest_rated most_liked"`
	Page      int    `form:"page,default=1" binding:"min=1"`
	PageSize  int    `form:"pageSize,default=10" binding:"min=1,max=50"`
	TimeFrame string `form:"timeFrame" binding:"omitempty,oneof=today this_week this_month all_time"`
}

func NewPlaceController(db *gorm.DB) *PlaceController {
	return &PlaceController{DB: db}
}

type SimplifiedPlace struct {
	ID         uint           `json:"id"`
	Name       string         `json:"name"`
	Categories pq.StringArray `json:"categories"`
	Address    string         `json:"address"`
	Latitude   float64        `json:"latitude"`
	Longitude  float64        `json:"longitude"`
	BaseScore  int            `json:"baseScore"`
	PlaceType  string         `json:"placeType"`
	PlaceImage string         `json:"placeImage"`
	IsVerified bool           `json:"isVerified"`
	Features   pq.StringArray `json:"features"`
}

// GetNearbyPlaces godoc
// @Summary Get nearby places based on location and zoom level with filters
// @Tags places
// @Accept json
// @Produce json
// @Param latitude query number true "User's latitude"
// @Param longitude query number true "User's longitude"
// @Param zoomLevel query integer true "Map zoom level (1-20)"
// @Param radius query number false "Search radius in kilometers"
// @Param hideVisited query boolean false "Hide places already visited by the user"
// @Param userId query integer false "User ID (required if hideVisited is true)"
// @Param category query string false "Filter by category"
// @Param maxPlaces query integer false "Maximum number of places to return"
// @Success 200 {object} types.NearbyPlacesResponse
// @Router /places/nearby [get]
func (pc *PlaceController) GetNearbyPlaces(c *gin.Context) {
	// Get user from context
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	var query NearbyPlacesQuery
	
	// Try to bind query parameters first
	if err := c.ShouldBindQuery(&query); err != nil {
		// If direct binding fails, try to parse nested params format
		query.Latitude = parseFloat(c.Query("params[latitude]"))
		query.Longitude = parseFloat(c.Query("params[longitude]"))
		query.ZoomLevel = parseInt(c.Query("params[zoomLevel]"))
		query.Radius = parseFloat(c.Query("params[radius]"))
		query.HideVisited = parseBool(c.Query("params[hideVisited]"))
		query.CategoryFilter = c.Query("params[category]")
		query.MaxPlaces = parseInt(c.Query("params[maxPlaces]"))
		
		// Validate required fields
		if query.Latitude == 0 || query.Longitude == 0 || query.ZoomLevel == 0 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "latitude, longitude, and zoomLevel are required",
				"debug": gin.H{
					"received_params": c.Request.URL.Query(),
					"parsed_values": gin.H{
						"latitude": query.Latitude,
						"longitude": query.Longitude,
						"zoomLevel": query.ZoomLevel,
					},
				},
			})
			return
		}
		
		// Validate zoom level range
		if query.ZoomLevel < 1 || query.ZoomLevel > 20 {
			c.JSON(http.StatusBadRequest, gin.H{
				"error": "zoomLevel must be between 1 and 20",
			})
			return
		}
	}

	// Use user-provided coordinates and radius
	latitude := query.Latitude
	longitude := query.Longitude

	// Default radius to 20km if not specified  
	radius := 20.0
	if query.Radius > 0 {
		radius = query.Radius / 1000.0 // Convert meters to kilometers
	}

	// Adjust radius based on zoom level (smaller radius for higher zoom)
	// Higher zoom = closer view = smaller radius
	zoomFactor := float64(query.ZoomLevel) / 20.0 // 0.05 to 1.0
	if zoomFactor > 1.0 {
		zoomFactor = 1.0
	}
	radius = radius * (2.0 - zoomFactor) // Max 2x radius at zoom 1, normal radius at zoom 20
	
	// Ensure minimum and maximum radius limits
	if radius < 0.1 { // Minimum 100m
		radius = 0.1
	}
	if radius > 50.0 { // Maximum 50km
		radius = 50.0
	}

	// Limit number of places to return
	limit := 50 // Default number of places to return
	if query.MaxPlaces > 0 && query.MaxPlaces < limit {
		limit = query.MaxPlaces
	}

	// Get points configuration
	pointsConfig := types.GetPointsConfig()
	
	// Build the query with conditional point_value based on user posts and no posts bonus
	db := pc.DB.Model(&models.Place{}).
		Select(`id, latitude, longitude, 
			CASE 
				WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
				THEN ? 
				WHEN NOT EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id)
				THEN base_points + ?
				ELSE base_points 
			END as point_value, 
			is_verified, 
			(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance`,
			user.UserID, pointsConfig.UserVisitedPoints, pointsConfig.NoPostsBonusPoints, latitude, longitude, latitude).
		Where("(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= ?",
			latitude, longitude, latitude, radius)

	// Apply category filter if provided
	if query.CategoryFilter != "" {
		db = db.Where("? = ANY(categories)", query.CategoryFilter)
	}

	// Order by distance and limit results
	db = db.Order("distance").Limit(limit)

	// Place bilgilerini hesaplanmış point_value ile çek
	var places []struct {
		ID         uint           `json:"id"`
		Latitude   float64        `json:"latitude"`
		Longitude  float64        `json:"longitude"`
		PointValue int            `json:"point_value"`
		IsVerified bool           `json:"is_verified"`
		Distance   float64        `json:"distance"`
		Categories pq.StringArray `json:"categories"`
	}
	
	result := db.Select(`id, latitude, longitude, 
		CASE 
			WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
			THEN ? 
			WHEN NOT EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id)
			THEN base_points + ?
			ELSE base_points 
		END as point_value, 
		is_verified, 
		(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance,
		categories`,
		user.UserID, pointsConfig.UserVisitedPoints, pointsConfig.NoPostsBonusPoints, latitude, longitude, latitude).Find(&places)
	
	// Markers'ı yarıçap bilgileriyle birlikte oluştur
	var markers []types.PlaceWithRadius
	for _, place := range places {
		postRadius, radiusType, radiusDescription, coverageArea := types.GetPlacePostRadius(place.Categories)
		
		marker := types.PlaceWithRadius{
			ID:                place.ID,
			Latitude:          place.Latitude,
			Longitude:         place.Longitude,
			PointValue:        place.PointValue,
			IsVerified:        place.IsVerified,
			Distance:          place.Distance,
			PostRadius:        postRadius,
			CoverageArea:      coverageArea,
			RadiusType:        radiusType,
			RadiusDescription: radiusDescription,
		}
		markers = append(markers, marker)
	}

	if result.RowsAffected < 20 {
		// Google Places API'den yeni yerler al ve kaydet
		log.Printf("Attempting to fetch places from Google Places API for location: %f,%f with radius: %f", latitude, longitude, radius)
		err := fetchAndSaveFromGooglePlaces(pc.DB, latitude, longitude, radius)
		if err != nil {
			log.Printf("Google Places API error: %v", err)
			// API hatası durumunda graceful fallback - mevcut verilerle devam et
			log.Printf("Falling back to existing data. Current markers count: %d", result.RowsAffected)
			
			if result.RowsAffected == 0 {
				// Hiç veri yoksa boş sonuç döndür ama başarılı response ver
				log.Printf("No existing markers found, returning empty result")
				response := types.NearbyPlacesResponse{
					Markers: []types.PlaceWithRadius{},
					Filters: struct {
						Radius      float64 `json:"radius"`
						ZoomLevel   int     `json:"zoomLevel"`
						HideVisited bool    `json:"hideVisited"`
						Category    string  `json:"category"`
					}{
						Radius:      radius,
						ZoomLevel:   query.ZoomLevel,
						HideVisited: query.HideVisited,
						Category:    query.CategoryFilter,
					},
				}
				c.JSON(http.StatusOK, response)
				return
			}
			// Mevcut verilerle devam et (API hatasını logla ama client'a hata dönme)
			log.Printf("Continuing with existing %d markers", result.RowsAffected)
		} else {
			// API başarılı olduğunda yeniden veritabanından güncel yerleri çek
			places = []struct {
				ID         uint           `json:"id"`
				Latitude   float64        `json:"latitude"`
				Longitude  float64        `json:"longitude"`
				PointValue int            `json:"point_value"`
				IsVerified bool           `json:"is_verified"`
				Distance   float64        `json:"distance"`
				Categories pq.StringArray `json:"categories"`
			}{}
			result = db.Select(`id, latitude, longitude, 
				CASE 
					WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
					THEN ? 
					WHEN NOT EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id)
					THEN base_points + ?
					ELSE base_points 
				END as point_value, 
				is_verified, 
				(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) AS distance,
				categories`,
				user.UserID, pointsConfig.UserVisitedPoints, pointsConfig.NoPostsBonusPoints, latitude, longitude, latitude).Find(&places)
			if result.Error != nil {
				c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching updated places"})
				return
			}
			
			// Güncellenmiş markers'ı oluştur
			markers = []types.PlaceWithRadius{}
			for _, place := range places {
				postRadius, radiusType, radiusDescription, coverageArea := types.GetPlacePostRadius(place.Categories)
				
				marker := types.PlaceWithRadius{
					ID:                place.ID,
					Latitude:          place.Latitude,
					Longitude:         place.Longitude,
					PointValue:        place.PointValue,
					IsVerified:        place.IsVerified,
					Distance:          place.Distance,
					PostRadius:        postRadius,
					CoverageArea:      coverageArea,
					RadiusType:        radiusType,
					RadiusDescription: radiusDescription,
				}
				markers = append(markers, marker)
			}
		}
	}
	

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching places"})
		return
	}

	response := types.NearbyPlacesResponse{
		Markers: markers,
		Filters: struct {
			Radius      float64 `json:"radius"`
			ZoomLevel   int     `json:"zoomLevel"`
			HideVisited bool    `json:"hideVisited"`
			Category    string  `json:"category"`
		}{
			Radius:      radius,
			ZoomLevel:   query.ZoomLevel,
			HideVisited: query.HideVisited,
			Category:    query.CategoryFilter,
		},
	}

	c.JSON(http.StatusOK, response)
}



func fetchAndSaveFromGooglePlaces(db *gorm.DB, lat, lng, radius float64) error {
	return fetchAndSaveFromGooglePlacesWithToken(db, lat, lng, radius, "", 0)
}

func fetchAndSaveFromGooglePlacesWithToken(db *gorm.DB, lat, lng, radius float64, pageToken string, pageCount int) error {
	// Maksimum sayfa sayısını sınırla (rate limiting için)
	if pageCount >= 3 {
		return nil
	}

	// Google Places API key kontrolü
	apiKey := os.Getenv("GOOGLE_PLACES_API_KEY")
	if apiKey == "" {
		return fmt.Errorf("Google Places API key not configured")
	}

	// Google Places API URL hazırla
	var url string
	if pageToken != "" {
		url = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?pagetoken=%s&key=%s", pageToken, apiKey)
	} else {
		// radius kilometre cinsinden geldiği için metre'ye çevir
		radiusInMeters := radius * 1000
		if radiusInMeters > 50000 { // Google API max 50km
			radiusInMeters = 50000
		}
		url = fmt.Sprintf("https://maps.googleapis.com/maps/api/place/nearbysearch/json?location=%f,%f&radius=%.0f&key=%s", lat, lng, radiusInMeters, apiKey)
	}

	// NextPageToken kullanıyorsak kısa bir bekleme süresi ekle (Google'ın önerisi)
	if pageToken != "" {
		time.Sleep(2 * time.Second)
	}

	fmt.Printf("Fetching page %d, URL: %s\n", pageCount+1, url)
	
	// HTTP GET isteği gönder
	resp, err := http.Get(url)
	if err != nil {
		return fmt.Errorf("error calling Google Places API: %w", err)
	}
	defer resp.Body.Close()

	// Cevabı çözümle
	var apiResponse types.GooglePlacesResponse

	if err := json.NewDecoder(resp.Body).Decode(&apiResponse); err != nil {
		return fmt.Errorf("error decoding API response: %w", err)
	}

	// API response status kontrolü
	if apiResponse.Status != "OK" && apiResponse.Status != "ZERO_RESULTS" {
		log.Printf("Google Places API error response: Status=%s, Results=%d", apiResponse.Status, len(apiResponse.Results))
		
		// Özel hata mesajları
		switch apiResponse.Status {
		case "REQUEST_DENIED":
			return fmt.Errorf("Google Places API access denied - check API key and permissions")
		case "OVER_QUERY_LIMIT":
			return fmt.Errorf("Google Places API query limit exceeded")
		case "INVALID_REQUEST":
			return fmt.Errorf("Google Places API invalid request parameters")
		default:
			return fmt.Errorf("Google Places API error: %s", apiResponse.Status)
		}
	}

	log.Printf("Fetched %d places from Google Places API (page %d)", len(apiResponse.Results), pageCount+1)

	// Mevcut yerleri çekme clustering için
	var existingPlaces []types.PlaceForClustering
	
	// GORM'dan direkt olarak PlaceForClustering struct'ına map etmek yerine
	// veritabanından raw model alıp dönüştür
	var existingPlaceModels []models.Place
	db.Select("latitude, longitude, categories, rating, name").
		Where("(6371 * acos(cos(radians(?)) * cos(radians(latitude)) * cos(radians(longitude) - radians(?)) + sin(radians(?)) * sin(radians(latitude)))) <= 10", lat, lng, lat).
		Find(&existingPlaceModels)

	// types.PlaceForClustering'e dönüştür
	for _, p := range existingPlaceModels {
		existingPlaces = append(existingPlaces, types.PlaceForClustering{
			Latitude:   p.Latitude,
			Longitude:  p.Longitude,
			Categories: []string(p.Categories),
			Rating:     p.Rating,
			Name:       p.Name,
		})
	}

	// Akıllı yer seçim algoritması - popüler ve dağıtılmış yerler seç
	candidatePlaces := make([]types.GooglePlaceResult, 0)
	
	// Önce tüm places'leri filtrele ve puanla
	for _, place := range apiResponse.Results {
		// 1. Temel filtreleme - mantıksız yerleri dışla
		if types.ShouldExcludePlace(place.Types, place.Name, place.Rating, place.UserRatingsTotal) {
			continue
		}
		
		candidatePlaces = append(candidatePlaces, place)
	}
	
	// Candidate places'leri popülerlik puanına göre sırala
	candidatePlaces = sortPlacesByImportance(candidatePlaces)
	
	// Akıllı seçim algoritması - dağıtım ve popülerlik dengesi
	selectedPlaces := selectBestDistributedPlaces(candidatePlaces, existingPlaces, 20) // Her sayfada maksimum 20 yer seç
	
	// Seçilen yerleri veritabanına kaydet
	savedCount := 0
	filteredCount := len(apiResponse.Results) - len(candidatePlaces)
	clusteredCount := len(candidatePlaces) - len(selectedPlaces)
	
	for _, place := range selectedPlaces {
		// Seçilen yeri mevcut listesine ekle
		existingPlaces = append(existingPlaces, types.PlaceForClustering{
			Latitude:   place.Geometry.Location.Lat,
			Longitude:  place.Geometry.Location.Lng,
			Categories: place.Types,
			Rating:     place.Rating,
			Name:       place.Name,
		})
		// Kategori bilgilerini al
		categories := pq.StringArray(place.Types)
		
		// Adres bilgisini vicinity'den al
		address := ""
		if place.Vicinity != nil {
			address = *place.Vicinity
		}

		// Fotoğraf referanslarını al
		var photoReferences pq.StringArray
		for _, photo := range place.Photos {
			photoReferences = append(photoReferences, photo.PhotoReference)
		}

		// Plus code bilgilerini al
		plusCode := ""
		if place.PlusCode != nil {
			plusCode = place.PlusCode.GlobalCode
		}

		// Business status kontrolü
		businessStatus := ""
		if place.BusinessStatus != nil {
			businessStatus = *place.BusinessStatus
		}

		// Gelişmiş puan hesaplama sistemi
		basePoints := types.CalculatePlacePoints(place.Types, place.Rating, place.UserRatingsTotal)

		// Handle opening hours - set to nil if not available
		var openingHours *string
		if place.OpeningHours != nil {
			// Convert opening hours to JSON string if available
			jsonStr := `{"periods":[],"weekday_text":[]}`
			openingHours = &jsonStr
		}

		dbPlace := models.Place{
			Name:              place.Name,
			Latitude:          place.Geometry.Location.Lat,
			Longitude:         place.Geometry.Location.Lng,
			Address:           address,
			PlaceType:         "google_place",
			Categories:        categories,
			BasePoints:        basePoints,
			GooglePlaceID:     place.PlaceID,
			Rating:            place.Rating,
			UserRatingsTotal:  place.UserRatingsTotal,
			BusinessStatus:    businessStatus,
			Icon:              place.Icon,
			PhotoReferences:   photoReferences,
			PlusCode:          plusCode,
			OpeningHours:      openingHours,
		}

		// Google Place ID ile çakışma varsa güncelle, yoksa ekle
		result := db.Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "google_place_id"}},
			DoUpdates: clause.AssignmentColumns([]string{
				"name", "latitude", "longitude", "address", "categories",
				"rating", "user_ratings_total", "business_status", "icon",
				"photo_references", "plus_code", "updated_at",
			}),
		}).Create(&dbPlace)
		
		if result.Error != nil {
			log.Printf("Insert/Update error for place %s: %v", place.Name, result.Error)
		} else {
			savedCount++
		}
	}

	log.Printf("Page %d results: %d total, %d filtered, %d clustered, %d saved using smart distribution algorithm", 
		pageCount+1, len(apiResponse.Results), filteredCount, clusteredCount, savedCount)

	// NextPageToken varsa ve daha fazla sayfa alınabiliyorsa, bir sonraki sayfayı al
	if apiResponse.NextPageToken != "" && pageCount < 2 {
		log.Printf("NextPageToken found, fetching next page...")
		return fetchAndSaveFromGooglePlacesWithToken(db, lat, lng, radius, apiResponse.NextPageToken, pageCount+1)
	}

	return nil
}

// GetPlaceProfile godoc
// @Summary Get detailed profile information about a place
// @Description Returns comprehensive place information including stats and recent activity
// @Tags places
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Success 200 {object} map[string]interface{}
// @Router /places/{placeId}/profile [get]
func (pc *PlaceController) GetPlaceProfile(c *gin.Context) {
	// Get user from context
	user := utils.GetUser(c)
	if user == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "User not found in context"})
		return
	}

	placeIdStr := c.Param("placeId")
	
	// Validate placeId parameter
	if placeIdStr == "" || placeIdStr == "undefined" || placeIdStr == "null" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid place ID"})
		return
	}
	
	// Convert to integer to ensure it's a valid ID
	placeId, err := strconv.Atoi(placeIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Place ID must be a valid number"})
		return
	}

	// Get points configuration
	pointsConfig := types.GetPointsConfig()
	
	// Place temel bilgileri - kullanıcının post atıp atmadığına göre point_value hesapla
	var place struct {
		ID                uint               `json:"id"`
		Name              string             `json:"name"`
		Latitude          float64            `json:"latitude"`
		Longitude         float64            `json:"longitude"`
		PointValue        int                `json:"point_value"`
		PlaceImage        string             `json:"place_image"`
		Categories        pq.StringArray     `json:"categories"`
		Address           string             `json:"address"`
		GooglePlaceID     string             `json:"google_place_id"`
		Rating            *float64           `json:"rating"`
		UserRatingsTotal  *int               `json:"user_ratings_total"`
		BusinessStatus    string             `json:"business_status"`
		Icon              string             `json:"icon"`
		PhotoReferences   pq.StringArray     `json:"photo_references"`
		PlusCode          string             `json:"plus_code"`
		Phone             string             `json:"phone"`
		Website           string             `json:"website"`
		PriceLevel        *int               `json:"price_level"`
		OpeningHours      *string            `json:"opening_hours"`
		PlaceType         string             `json:"place_type"`
		IsVerified        bool               `json:"is_verified"`
		Features          pq.StringArray     `json:"features"`
	}
	
	// First get the basic place data
	var placeModel models.Place
	if err := pc.DB.Where("id = ?", placeId).First(&placeModel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Calculate point value separately
	var pointValue int
	if err := pc.DB.Model(&models.Place{}).
		Select(`CASE 
			WHEN EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id AND posts.user_id = ?) 
			THEN ? 
			WHEN NOT EXISTS(SELECT 1 FROM posts WHERE posts.place_id = places.id)
			THEN base_points + ?
			ELSE base_points 
		END as point_value`, user.UserID, pointsConfig.UserVisitedPoints, pointsConfig.NoPostsBonusPoints).
		Where("id = ?", placeId).
		Scan(&pointValue).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Map to our response struct
	place.ID = placeModel.ID
	place.Name = placeModel.Name
	place.Latitude = placeModel.Latitude
	place.Longitude = placeModel.Longitude
	place.PointValue = pointValue
	place.PlaceImage = placeModel.PlaceImage
	place.Categories = placeModel.Categories
	place.Address = placeModel.Address
	place.GooglePlaceID = placeModel.GooglePlaceID
	place.Rating = placeModel.Rating
	place.UserRatingsTotal = placeModel.UserRatingsTotal
	place.BusinessStatus = placeModel.BusinessStatus
	place.Icon = placeModel.Icon
	place.PhotoReferences = placeModel.PhotoReferences
	place.PlusCode = placeModel.PlusCode
	place.Phone = placeModel.Phone
	place.Website = placeModel.Website
	place.PriceLevel = placeModel.PriceLevel
	place.OpeningHours = placeModel.OpeningHours
	place.PlaceType = placeModel.PlaceType
	place.IsVerified = placeModel.IsVerified
	place.Features = placeModel.Features

	// Stat bilgileri
	var stats struct {
		TotalPosts    int64     `json:"totalPosts"`
		TotalPoints   int64     `json:"totalPoints"`
		UniquePosters int64     `json:"uniquePosters"`
		LastPostTime  time.Time `json:"lastPostTime"`
	}
	pc.DB.Model(&models.Post{}).Where("place_id = ?", placeId).Count(&stats.TotalPosts)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", placeId).Select("COALESCE(SUM(earned_points), 0)").Scan(&stats.TotalPoints)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", placeId).Distinct("user_id").Count(&stats.UniquePosters)
	pc.DB.Model(&models.Post{}).Where("place_id = ?", placeId).Select("COALESCE(MAX(created_at), ?)", time.Time{}).Scan(&stats.LastPostTime)

	// Kullanıcıları grupla - her kullanıcının kaç post attığını göster
	var userPosts []struct {
		UserID      uint      `json:"userId"`
		Username    string    `json:"username"`
		FirstName   string    `json:"firstName"`
		LastName    string    `json:"lastName"`
		Avatar      string    `json:"avatar"`
		PostCount   int64     `json:"postCount"`
		TotalPoints int64     `json:"totalPoints"`
		LastPostAt  time.Time `json:"lastPostAt"`
	}

	pc.DB.Table("posts").
		Select(`
			users.id as user_id,
			users.username,
			users.first_name,
			users.last_name,
			users.avatar,
			COUNT(posts.id) as post_count,
			COALESCE(SUM(posts.earned_points), 0) as total_points,
			MAX(posts.created_at) as last_post_at
		`).
		Joins("JOIN users ON users.id = posts.user_id").
		Where("posts.place_id = ?", placeId).
		Group("users.id, users.username, users.first_name, users.last_name, users.avatar").
		Order("post_count DESC, last_post_at DESC").
		Find(&userPosts)

	// En çok post atan ilk 5 kullanıcıyı getir (top users için ayrı)
	var topUsers []struct {
		UserID      uint   `json:"id"`
		Username    string `json:"username"`
		FirstName   string `json:"first_name"`
		LastName    string `json:"last_name"`
		TotalPoints int64  `json:"total_points"`
		PostCount   int64  `json:"post_count"`
		Avatar      string `json:"avatar"`
	}
	pc.DB.Table("posts").
		Select("user_id as user_id, users.username, users.first_name, users.last_name, users.total_points, users.avatar, COUNT(posts.id) as post_count").
		Joins("JOIN users ON users.id = posts.user_id").
		Where("place_id = ?", placeId).
		Group("user_id, users.username, users.first_name, users.last_name, users.total_points, users.avatar").
		Order("post_count DESC").
		Limit(5).
		Scan(&topUsers)

	response := gin.H{
		"id":                 place.ID,
		"name":               place.Name,
		"latitude":           place.Latitude,
		"longitude":          place.Longitude,
		"point_value":        place.PointValue,
		"place_image":        place.PlaceImage,
		"categories":         place.Categories,
		"address":            place.Address,
		"google_place_id":    place.GooglePlaceID,
		"rating":             place.Rating,
		"user_ratings_total": place.UserRatingsTotal,
		"business_status":    place.BusinessStatus,
		"icon":               place.Icon,
		"photo_references":   place.PhotoReferences,
		"plus_code":          place.PlusCode,
		"phone":              place.Phone,
		"website":            place.Website,
		"price_level":        place.PriceLevel,
		"opening_hours":      place.OpeningHours,
		"place_type":         place.PlaceType,
		"is_verified":        place.IsVerified,
		"features":           place.Features,
		"stats":              stats,
		"user_posts":         userPosts,
		"top_users":          topUsers,
	}

	c.JSON(http.StatusOK, response)
}

// GetPlacePosts godoc
// @Summary Get posts from a specific place with sorting and pagination
// @Description Returns paginated posts from a place with various sorting options
// @Tags places
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Param sortBy query string false "Sort posts by: newest, highest_rated, most_liked"
// @Param page query integer false "Page number (default: 1)"
// @Param pageSize query integer false "Items per page (default: 10, max: 50)"
// @Param timeFrame query string false "Time frame: today, this_week, this_month, all_time"
// @Success 200 {object} map[string]interface{}
// @Router /places/{placeId}/posts [get]
func (pc *PlaceController) GetPlacePosts(c *gin.Context) {
	placeIdStr := c.Param("placeId")
	
	// Validate placeId parameter
	if placeIdStr == "" || placeIdStr == "undefined" || placeIdStr == "null" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid place ID"})
		return
	}
	
	// Convert to integer to ensure it's a valid ID
	placeId, err := strconv.Atoi(placeIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Place ID must be a valid number"})
		return
	}
	
	var query PlacePostsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	db := pc.DB.Model(&models.Post{}).Where("place_id = ?", placeId)

	// Apply time frame filter
	switch query.TimeFrame {
	case "today":
		db = db.Where("created_at >= CURRENT_DATE")
	case "this_week":
		db = db.Where("created_at >= DATE_TRUNC('week', CURRENT_DATE)")
	case "this_month":
		db = db.Where("created_at >= DATE_TRUNC('month', CURRENT_DATE)")
	}

	// Apply sorting
	switch query.SortBy {
	case "highest_rated":
		db = db.Order("points DESC")
	case "most_liked":
		db = db.Order("(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) DESC")
	default: // "newest" or empty
		db = db.Order("created_at DESC")
	}

	// Calculate pagination
	offset := (query.Page - 1) * query.PageSize

	var total int64
	db.Count(&total)

	var posts []struct {
		models.Post
		LikesCount    int64  `json:"likesCount"`
		CommentsCount int64  `json:"commentsCount"`
		Username      string `json:"username"`
	}

	result := db.
		Select("posts.*, users.username, " +
			"(SELECT COUNT(*) FROM likes WHERE likes.post_id = posts.id) as likes_count, " +
			"(SELECT COUNT(*) FROM comments WHERE comments.post_id = posts.id) as comments_count").
		Joins("JOIN users ON users.id = posts.user_id").
		Offset(offset).
		Limit(query.PageSize).
		Find(&posts)

	if result.Error != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "Error fetching posts"})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"posts": posts,
		"pagination": gin.H{
			"currentPage": query.Page,
			"pageSize":    query.PageSize,
			"totalItems":  total,
			"totalPages":  math.Ceil(float64(total) / float64(query.PageSize)),
		},
	})
}

// ValidatePostLocation godoc
// @Summary Validate if user is within the allowed radius to post at a place
// @Tags places
// @Accept json
// @Produce json
// @Param placeId path string true "Place ID"
// @Param latitude query number true "User's current latitude"
// @Param longitude query number true "User's current longitude"
// @Success 200 {object} map[string]interface{}
// @Router /places/{placeId}/validate-location [get]
func (pc *PlaceController) ValidatePostLocation(c *gin.Context) {
	placeIdStr := c.Param("placeId")
	
	// Validate placeId parameter
	if placeIdStr == "" || placeIdStr == "undefined" || placeIdStr == "null" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid place ID"})
		return
	}
	
	// Convert to integer to ensure it's a valid ID
	placeId, err := strconv.Atoi(placeIdStr)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Place ID must be a valid number"})
		return
	}

	// Get user coordinates
	userLatStr := c.Query("latitude")
	userLngStr := c.Query("longitude")
	
	if userLatStr == "" || userLngStr == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "User latitude and longitude are required"})
		return
	}

	userLat, err := strconv.ParseFloat(userLatStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid latitude format"})
		return
	}

	userLng, err := strconv.ParseFloat(userLngStr, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid longitude format"})
		return
	}

	// Get place information using the actual model to avoid pq.StringArray issues
	var placeModel models.Place
	if err := pc.DB.Select("id, name, latitude, longitude, categories").
		Where("id = ?", placeId).
		First(&placeModel).Error; err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Place not found"})
		return
	}

	// Calculate distance between user and place
	distance := types.CalculateDistance(userLat, userLng, placeModel.Latitude, placeModel.Longitude)
	distanceMeters := distance * 1000 // Convert to meters

	// Get place post radius
	postRadius, radiusType, radiusDescription, coverageArea := types.GetPlacePostRadius(placeModel.Categories)

	// Debug logging
	log.Printf("ValidatePostLocation - Place: %s, User: (%.6f,%.6f), Place: (%.6f,%.6f), Distance: %.2fm, Required: %dm, Categories: %v", 
		placeModel.Name, userLat, userLng, placeModel.Latitude, placeModel.Longitude, distanceMeters, postRadius, placeModel.Categories)

	// Check if user is within allowed radius
	isWithinRadius := distanceMeters <= float64(postRadius)

	response := gin.H{
		"place_id":            placeModel.ID,
		"place_name":          placeModel.Name,
		"user_latitude":       userLat,
		"user_longitude":      userLng,
		"place_latitude":      placeModel.Latitude,
		"place_longitude":     placeModel.Longitude,
		"distance_meters":     int(distanceMeters),
		"post_radius":         postRadius,
		"coverage_area":       coverageArea,
		"radius_type":         radiusType,
		"radius_description":  radiusDescription,
		"is_within_radius":    true,
		"categories":          placeModel.Categories,
		"can_post":            true,
	}
	//"is_within_radius":    isWithinRadius,
	//"can_post":            isWithinRadius,

	// Always return 200 with detailed information for debugging
	if !isWithinRadius {
		response["error"] = "You are too far from this place to post"
		response["required_distance"] = postRadius
		response["your_distance"] = int(distanceMeters)
		response["distance_difference"] = int(distanceMeters) - postRadius
	}
	
	c.JSON(http.StatusOK, response)
}

// Helper functions for parsing query parameters
func parseFloat(s string) float64 {
	if s == "" {
		return 0
	}
	val, err := strconv.ParseFloat(s, 64)
	if err != nil {
		return 0
	}
	return val
}

func parseInt(s string) int {
	if s == "" {
		return 0
	}
	val, err := strconv.Atoi(s)
	if err != nil {
		return 0
	}
	return val
}

func parseBool(s string) bool {
	if s == "" {
		return false
	}
	val, err := strconv.ParseBool(s)
	if err != nil {
		return false
	}
	return val
}

// Helper function to sort places by importance (popularity + rating + category significance)
func sortPlacesByImportance(places []types.GooglePlaceResult) []types.GooglePlaceResult {
	// Create a copy to avoid modifying the original slice
	sortedPlaces := make([]types.GooglePlaceResult, len(places))
	copy(sortedPlaces, places)
	
	// Sort by importance score (descending)
	for i := 0; i < len(sortedPlaces)-1; i++ {
		for j := i + 1; j < len(sortedPlaces); j++ {
			scoreI := calculatePlaceImportanceScore(sortedPlaces[i])
			scoreJ := calculatePlaceImportanceScore(sortedPlaces[j])
			
			if scoreI < scoreJ {
				sortedPlaces[i], sortedPlaces[j] = sortedPlaces[j], sortedPlaces[i]
			}
		}
	}
	
	return sortedPlaces
}

// Calculate importance score for a place
func calculatePlaceImportanceScore(place types.GooglePlaceResult) float64 {
	score := 0.0
	
	// Rating contribution (0-50 points)
	if place.Rating != nil {
		score += (*place.Rating - 3.0) * 10 // 3.0 = 0 points, 5.0 = 20 points
	}
	
	// Popularity contribution (0-50 points)
	if place.UserRatingsTotal != nil {
		switch {
		case *place.UserRatingsTotal >= 1000:
			score += 50
		case *place.UserRatingsTotal >= 500:
			score += 40
		case *place.UserRatingsTotal >= 200:
			score += 30
		case *place.UserRatingsTotal >= 100:
			score += 25
		case *place.UserRatingsTotal >= 50:
			score += 20
		case *place.UserRatingsTotal >= 20:
			score += 15
		case *place.UserRatingsTotal >= 10:
			score += 10
		default:
			score += 5
		}
	}
	
	// Category significance (0-30 points)
	for _, category := range place.Types {
		switch strings.ToLower(category) {
		case "tourist_attraction", "museum", "historical_site", "natural_feature":
			score += 30
		case "park", "restaurant", "shopping_mall", "theater":
			score += 20
		case "cafe", "store", "gym":
			score += 10
		default:
			score += 5
		}
		break // Only consider the first significant category
	}
	
	return score
}

// Select best distributed places using grid-based algorithm
func selectBestDistributedPlaces(candidates []types.GooglePlaceResult, existing []types.PlaceForClustering, maxPlaces int) []types.GooglePlaceResult {
	if len(candidates) == 0 {
		return []types.GooglePlaceResult{}
	}
	
	selected := make([]types.GooglePlaceResult, 0, maxPlaces)
	
	// Grid-based selection to ensure good distribution
	const gridSize = 0.01 // ~1km grid cells
	occupiedCells := make(map[string]bool)
	
	// Mark existing places' grid cells as occupied
	for _, place := range existing {
		cellKey := fmt.Sprintf("%.2f,%.2f", 
			math.Floor(place.Latitude/gridSize)*gridSize,
			math.Floor(place.Longitude/gridSize)*gridSize)
		occupiedCells[cellKey] = true
	}
	
	// First pass: Select highly important places regardless of distribution
	highImportanceThreshold := 80.0
	for _, place := range candidates {
		if len(selected) >= maxPlaces {
			break
		}
		
		importance := calculatePlaceImportanceScore(place)
		if importance >= highImportanceThreshold {
			selected = append(selected, place)
			
			// Mark this cell as occupied
			cellKey := fmt.Sprintf("%.2f,%.2f", 
				math.Floor(place.Geometry.Location.Lat/gridSize)*gridSize,
				math.Floor(place.Geometry.Location.Lng/gridSize)*gridSize)
			occupiedCells[cellKey] = true
		}
	}
	
	// Second pass: Fill remaining slots with distributed places
	for _, place := range candidates {
		if len(selected) >= maxPlaces {
			break
		}
		
		// Skip if already selected
		alreadySelected := false
		for _, sel := range selected {
			if sel.PlaceID == place.PlaceID {
				alreadySelected = true
				break
			}
		}
		if alreadySelected {
			continue
		}
		
		// Check if this grid cell is already occupied
		cellKey := fmt.Sprintf("%.2f,%.2f", 
			math.Floor(place.Geometry.Location.Lat/gridSize)*gridSize,
			math.Floor(place.Geometry.Location.Lng/gridSize)*gridSize)
		
		if !occupiedCells[cellKey] {
			selected = append(selected, place)
			occupiedCells[cellKey] = true
		}
	}
	
	// Third pass: Fill any remaining slots with best remaining places
	for _, place := range candidates {
		if len(selected) >= maxPlaces {
			break
		}
		
		// Skip if already selected
		alreadySelected := false
		for _, sel := range selected {
			if sel.PlaceID == place.PlaceID {
				alreadySelected = true
				break
			}
		}
		if alreadySelected {
			continue
		}
		
		// Check minimum distance to avoid too close places
		tooClose := false
		for _, sel := range selected {
			distance := types.CalculateDistance(
				place.Geometry.Location.Lat, place.Geometry.Location.Lng,
				sel.Geometry.Location.Lat, sel.Geometry.Location.Lng)
			if distance < 0.2 { // 200m minimum distance
				tooClose = true
				break
			}
		}
		
		if !tooClose {
			selected = append(selected, place)
		}
	}
	
	return selected
}
