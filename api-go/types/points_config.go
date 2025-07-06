package types

import (
	"math"
	"strings"
)

const (
	DEFAULT_PLACE_POINTS = 5
	USER_VISITED_POINTS = 1
	NO_POSTS_BONUS_POINTS = 3
)

type PointsConfig struct {
	DefaultPlacePoints  int
	UserVisitedPoints   int
	NoPostsBonusPoints  int
}

type PlaceScoring struct {
	CategoryPoints    map[string]int
	RarityMultiplier  map[string]float64
	PopularityBonus   map[string]int
	RatingBonus       map[string]int
}

type PlaceFiltering struct {
	ExcludedCategories []string
	MinimumDistance    float64
	QualityThresholds  map[string]QualityThreshold
}

type QualityThreshold struct {
	MinRating           float64
	MinUserRatingsTotal int
	RequiredKeywords    []string
	ExcludedKeywords    []string
}

type PlaceRadius struct {
	CategoryRadius map[string]int // metre cinsinden
	DefaultRadius  int
}

type PlaceWithRadius struct {
	ID                  uint           `json:"id"`
	Latitude            float64        `json:"latitude"`
	Longitude           float64        `json:"longitude"`
	PointValue          int            `json:"point_value"`
	IsVerified          bool           `json:"is_verified"`
	Distance            float64        `json:"distance"`
	PostRadius          int            `json:"post_radius"`          // Post atabilmek için gerekli yarıçap (metre)
	CoverageArea        float64        `json:"coverage_area"`        // Yerin kapladığı alan (m²)
	RadiusType          string         `json:"radius_type"`          // Programatik key (small, medium, large, etc.)
	RadiusDescription   string         `json:"radius_description"`   // İnsan dostu açıklama
}

func GetPointsConfig() PointsConfig {
	return PointsConfig{
		DefaultPlacePoints:  DEFAULT_PLACE_POINTS,
		UserVisitedPoints:   USER_VISITED_POINTS,
		NoPostsBonusPoints:  NO_POSTS_BONUS_POINTS,
	}
}

func GetPlaceRadius() PlaceRadius {
	return PlaceRadius{
		CategoryRadius: map[string]int{
			// Çok büyük alanlar - geniş yarıçap
			"national_park": 1000,     // 1km
			"state_park": 800,         // 800m
			"regional_park": 500,      // 500m
			"country_park": 400,       // 400m
			"forest": 600,             // 600m
			"natural_feature": 300,    // 300m
			"mountain": 800,           // 800m
			"lake": 400,               // 400m
			"beach": 300,              // 300m
			"island": 500,             // 500m
			"valley": 400,             // 400m
			"desert": 600,             // 600m
			
			// Büyük yapılar ve kompleksler
			"university": 400,         // 400m - kampüs alanı
			"hospital": 200,           // 200m - hastane kompleksi
			"airport": 800,            // 800m - havalimanı
			"train_station": 150,      // 150m - tren istasyonu
			"stadium": 200,            // 200m - stadyum
			"convention_center": 200,  // 200m
			"exhibition_center": 200,  // 200m
			"fairground": 300,         // 300m
			"race_track": 400,         // 400m
			
			// Tarihi ve kültürel alanlar
			"castle": 300,             // 300m - kale kompleksi
			"palace": 250,             // 250m - saray
			"historical_site": 200,    // 200m
			"archaeological_site": 250, // 250m
			"ruins": 150,              // 150m
			"monument": 50,            // 50m
			"memorial": 30,            // 30m
			
			// Büyük parklar ve bahçeler
			"park": 200,               // 200m - şehir parkı
			"botanical_garden": 250,   // 250m
			"zoo": 300,                // 300m
			"safari_park": 500,        // 500m
			"theme_park": 400,         // 400m
			"amusement_park": 300,     // 300m
			"water_park": 200,         // 200m
			
			// Alışveriş merkezleri
			"shopping_mall": 150,      // 150m - AVM
			"shopping_center": 100,    // 100m
			"market": 80,              // 80m
			"bazaar": 100,             // 100m
			
			// Dini yapılar
			"mosque": 100,             // 100m - büyük camiler
			"church": 80,              // 80m
			"cathedral": 150,          // 150m - büyük katedraller
			"temple": 100,             // 100m
			"synagogue": 60,           // 60m
			"shrine": 50,              // 50m
			
			// Müzeler ve galeriler
			"museum": 120,             // 120m - müze binası
			"art_gallery": 80,         // 80m
			"science_museum": 150,     // 150m
			"history_museum": 120,     // 120m
			"aquarium": 150,           // 150m
			"planetarium": 80,         // 80m
			
			// Eğlence yerleri
			"movie_theater": 50,       // 50m - sinema
			"theater": 60,             // 60m - tiyatro
			"concert_hall": 80,        // 80m
			"opera_house": 100,        // 100m
			"night_club": 40,          // 40m
			"bar": 30,                 // 30m
			"pub": 40,                 // 40m
			"casino": 100,             // 100m
			
			// Spor tesisleri
			"gym": 50,                 // 50m
			"sports_complex": 200,     // 200m
			"swimming_pool": 80,       // 80m
			"golf_course": 300,        // 300m
			"tennis_court": 30,        // 30m
			"basketball_court": 25,    // 25m
			"football_field": 100,     // 100m
			"baseball_field": 80,      // 80m
			
			// Konaklama
			"hotel": 80,               // 80m - otel binası
			"resort": 200,             // 200m - tatil köyü
			"hostel": 40,              // 40m
			"motel": 50,               // 50m
			"bed_and_breakfast": 30,   // 30m
			"campground": 150,         // 150m
			
			// Restoran ve kafeler
			"restaurant": 25,          // 25m - restoran
			"cafe": 20,                // 20m - kafe
			"fast_food": 15,           // 15m
			"bakery": 15,              // 15m
			"food_court": 50,          // 50m
			"brewery": 40,             // 40m
			"winery": 100,             // 100m
			
			// Küçük işletmeler ve mağazalar
			"store": 20,               // 20m
			"clothing_store": 15,      // 15m
			"book_store": 20,          // 20m
			"jewelry_store": 10,       // 10m
			"electronics_store": 25,   // 25m
			"furniture_store": 30,     // 30m
			"hardware_store": 25,      // 25m
			"pharmacy": 15,            // 15m
			"supermarket": 40,         // 40m
			
			// Hizmet yerleri
			"bank": 20,                // 20m
			"post_office": 25,         // 25m
			"library": 60,             // 60m - kütüphane
			"school": 150,             // 150m - okul bahçesi
			"kindergarten": 50,        // 50m
			
			// Ulaşım
			"bus_station": 80,         // 80m
			"subway_station": 40,      // 40m
			"taxi_stand": 10,          // 10m
			"parking": 30,             // 30m
			"gas_station": 30,         // 30m
			
			// Genel kategoriler
			"tourist_attraction": 100, // 100m - genel turist yeri
			"point_of_interest": 50,   // 50m
			"establishment": 30,       // 30m
		},
		DefaultRadius: 25, // Varsayılan 25 metre
	}
}

func GetPlacePostRadius(categories []string) (int, string, string, float64) {
	radiusConfig := GetPlaceRadius()
	maxRadius := radiusConfig.DefaultRadius
	radiusType := "small"
	radiusDescription := "Küçük Alan"
	
	// En büyük yarıçapı bul
	for _, category := range categories {
		categoryLower := strings.ToLower(category)
		if radius, exists := radiusConfig.CategoryRadius[categoryLower]; exists && radius > maxRadius {
			maxRadius = radius
		}
	}
	
	// Yarıçap tipini belirle (hem key hem description)
	switch {
	case maxRadius >= 500:
		radiusType = "very_large"
		radiusDescription = "Çok Geniş Alan"
	case maxRadius >= 200:
		radiusType = "large"
		radiusDescription = "Geniş Alan"
	case maxRadius >= 100:
		radiusType = "medium"
		radiusDescription = "Orta Alan"
	case maxRadius >= 50:
		radiusType = "small_medium"
		radiusDescription = "Küçük-Orta Alan"
	default:
		radiusType = "small"
		radiusDescription = "Küçük Alan"
	}
	
	// Kapladığı alanı hesapla (π * r²)
	coverageArea := math.Pi * float64(maxRadius) * float64(maxRadius)
	
	return maxRadius, radiusType, radiusDescription, coverageArea
}

func GetPlaceFiltering() PlaceFiltering {
	return PlaceFiltering{
		ExcludedCategories: []string{
			// Servis sağlayıcıları - sosyal medya için uygun değil
			"locksmith",
			"plumber", 
			"electrician",
			"roofing_contractor",
			"general_contractor",
			"painter",
			"moving_company",
			"car_repair",
			"car_wash",
			"car_dealer",
			"gas_station",
			
			// Kişisel bakım - çok yaygın ve özel
			"hair_care",
			"beauty_salon",
			"spa",
			"nail_salon",
			"massage",
			"dentist",
			"doctor",
			"veterinary_care",
			"pharmacy",
			"physiotherapist",
			
			// Finansal hizmetler
			"atm",
			"bank",
			"insurance_agency",
			"accounting",
			"real_estate_agency",
			
			// Rutin hizmetler
			"laundry",
			"dry_cleaning",
			"post_office",
			"courier_service",
			"storage",
			
			// Çok spesifik işletmeler
			"funeral_home",
			"cemetery",
			"lawyer",
			"government_office",
			"courthouse",
			"police",
			"fire_station",
			
			// Alışveriş - çok yaygın olanlar
			"convenience_store",
			"supermarket",
			"grocery_or_supermarket",
			"hardware_store",
			"auto_parts_store",
			
			// Ulaşım altyapısı
			"parking",
			"taxi_stand",
			"bus_station",
			"subway_station",
			"truck_stop",
		},
		
		MinimumDistance: 50.0, // Minimum 50 metre mesafe
		
		QualityThresholds: map[string]QualityThreshold{
			"restaurant": {
				MinRating:           3.5,
				MinUserRatingsTotal: 10,
				ExcludedKeywords:    []string{"take", "takeaway", "delivery", "fast food", "drive"},
			},
			"cafe": {
				MinRating:           3.5,
				MinUserRatingsTotal: 5,
				ExcludedKeywords:    []string{"takeaway", "delivery"},
			},
			"store": {
				MinRating:           3.0,
				MinUserRatingsTotal: 5,
				RequiredKeywords:    []string{"boutique", "gallery", "art", "antique", "specialty"},
			},
			"lodging": {
				MinRating:           3.0,
				MinUserRatingsTotal: 10,
			},
			"bar": {
				MinRating:           3.5,
				MinUserRatingsTotal: 15,
				RequiredKeywords:    []string{"restaurant", "rooftop", "cocktail", "wine", "pub"},
			},
		},
	}
}

func GetPlaceScoring() PlaceScoring {
	return PlaceScoring{
		CategoryPoints: map[string]int{
			// Çok nadir ve özel yerler (60 puan - en nadir)
			"castle": 60,
			"palace": 60,
			"historical_site": 60, // UNESCO gibi özel siteler
			
			// Nadir kültürel yerler (55 puan)
			"museum": 55,
			"ruins": 55,
			
			// Önemli kültürel yerler (50 puan)
			"art_gallery": 50,
			"monument": 50,
			"archaeological_site": 50,
			
			// Popüler turistik yerler (45 puan)
			"tourist_attraction": 45,
			"natural_feature": 45,
			"waterfall": 45,
			"island": 45,
			
			// İyi yerler (40 puan)
			"church": 40,
			"mosque": 40,
			"synagogue": 40,
			"place_of_worship": 40,
			"national_park": 40,
			"mountain": 40,
			"cave": 40,
			
			// Orta kalite yerler (35 puan)
			"theater": 35,
			"beach": 35,
			"botanical_garden": 35,
			
			// Yaygın ama iyi yerler (30 puan)
			"park": 30,
			"zoo": 30,
			"aquarium": 30,
			"cemetery": 30,
			"amusement_park": 30,
			
			// Yaygın yerler (25 puan)
			"lake": 25,
			"forest": 25,
			"stadium": 25,
			"shopping_mall": 25,
			
			// Sıradan yerler (20 puan)
			"restaurant": 20,
			"movie_theater": 20,
			"spa": 20,
			
			// Çok yaygın yerler (15 puan - varsayılan)
			"cafe": 15,
			"night_club": 15,
			"casino": 15,
			"bar": 15,
			"hotel": 15,
			"lodging": 15,
			
			// Düşük puan yerler (10 puan)
			"gym": 10,
			"bowling_alley": 10,
			"bakery": 10,
			"food": 10,
			"meal_takeaway": 10,
			"meal_delivery": 10,
			
			// Özel alışveriş yerleri (15 puan)
			"book_store": 15,
			"jewelry_store": 15,
			
			// Sıradan alışveriş (10 puan)
			"store": 10,
			"clothing_store": 10,
			"electronics_store": 10,
			"supermarket": 10,
			
			// Özel konaklama (25-30 puan)
			"resort": 30,
			"hostel": 15,
			
			// Eğitim kurumları (20-25 puan)
			"university": 25,
			"library": 20,
			"school": 15,
			
			// Sağlık ve hizmetler (10-15 puan)
			"hospital": 15,
			"pharmacy": 10,
			"bank": 10,
			"post_office": 10,
			
			// Ulaşım merkezleri (10-15 puan)
			"train_station": 15,
			"airport": 15,
			"subway_station": 10,
			"bus_station": 10,
			
			// Çok düşük puan hizmetler (10 puan minimum)
			"gas_station": 10,
			"police": 10,
			"fire_station": 10,
			"parking": 10,
			"taxi_stand": 10,
			"atm": 10,
			
			// Genel kategoriler
			"establishment": 10,
			"point_of_interest": 15,
		},
		
		RarityMultiplier: map[string]float64{
			"very_rare": 2.0,    // Çok nadir yerler (örn: UNESCO siteleri)
			"rare": 1.5,         // Nadir yerler
			"uncommon": 1.2,     // Az bulunan yerler
			"common": 1.0,       // Yaygın yerler
			"very_common": 0.8,  // Çok yaygın yerler
		},
		
		PopularityBonus: map[string]int{
			"very_high": 15,  // Çok popüler (1000+ rating)
			"high": 10,       // Popüler (500-999 rating)
			"medium": 5,      // Orta (100-499 rating)
			"low": 0,         // Düşük (10-99 rating)
			"very_low": -5,   // Çok düşük (<10 rating)
		},
		
		RatingBonus: map[string]int{
			"excellent": 10,  // 4.5+ rating
			"very_good": 8,   // 4.0-4.4 rating
			"good": 5,        // 3.5-3.9 rating
			"average": 0,     // 3.0-3.4 rating
			"poor": -5,       // <3.0 rating
		},
	}
}

func ShouldExcludePlace(categories []string, name string, rating *float64, userRatingsTotal *int) bool {
	filtering := GetPlaceFiltering()
	
	// 1. Kategori bazlı filtreleme
	for _, category := range categories {
		categoryLower := strings.ToLower(category)
		for _, excluded := range filtering.ExcludedCategories {
			if categoryLower == excluded {
				return true
			}
		}
	}
	
	// 2. İsim bazlı filtreleme - şüpheli kelimeler
	nameLower := strings.ToLower(name)
	suspiciousKeywords := []string{
		"berber", "kuaför", "barber", "hair", "nail", "massage",
		"eczane", "pharmacy", "doktor", "doctor", "diş", "dental",
		"atm", "bank", "banka", "sigorta", "insurance",
		"benzin", "petrol", "gas station", "oto", "car wash",
		"tamirci", "repair", "servis", "service", "teknisyen",
		"kurye", "courier", "kargo", "cargo", "nakliye",
		"emlak", "real estate", "noter", "avukat", "lawyer",
		"muhasebe", "accounting", "mali müşavir",
		"temizlik", "cleaning", "dry clean", "laundry",
		"funeral", "cenaze", "mezar", "cemetery",
	}
	
	for _, keyword := range suspiciousKeywords {
		if strings.Contains(nameLower, keyword) {
			return true
		}
	}
	
	// 3. Kalite bazlı filtreleme
	for _, category := range categories {
		categoryLower := strings.ToLower(category)
		if threshold, exists := filtering.QualityThresholds[categoryLower]; exists {
			
			// Rating kontrolü
			if rating != nil && *rating < threshold.MinRating {
				return true
			}
			
			// Minimum değerlendirme sayısı kontrolü
			if userRatingsTotal != nil && *userRatingsTotal < threshold.MinUserRatingsTotal {
				return true
			}
			
			// Gerekli anahtar kelimeler
			if len(threshold.RequiredKeywords) > 0 {
				hasRequired := false
				for _, required := range threshold.RequiredKeywords {
					if strings.Contains(nameLower, strings.ToLower(required)) {
						hasRequired = true
						break
					}
				}
				if !hasRequired {
					return true
				}
			}
			
			// Dışlanacak anahtar kelimeler
			for _, excluded := range threshold.ExcludedKeywords {
				if strings.Contains(nameLower, strings.ToLower(excluded)) {
					return true
				}
			}
		}
	}
	
	return false
}

func CalculateDistance(lat1, lng1, lat2, lng2 float64) float64 {
	// Haversine formula ile mesafe hesaplama (kilometre)
	const R = 6371.0 // Earth radius in kilometers
	const PI = 3.14159265359
	
	// Convert degrees to radians
	lat1Rad := lat1 * PI / 180.0
	lng1Rad := lng1 * PI / 180.0
	lat2Rad := lat2 * PI / 180.0
	lng2Rad := lng2 * PI / 180.0
	
	dlat := lat2Rad - lat1Rad
	dlng := lng2Rad - lng1Rad
	
	// Haversine formula
	a := math.Sin(dlat/2)*math.Sin(dlat/2) + math.Cos(lat1Rad)*math.Cos(lat2Rad)*math.Sin(dlng/2)*math.Sin(dlng/2)
	c := 2 * math.Atan2(math.Sqrt(a), math.Sqrt(1-a))
	return R * c
}

func ShouldClusterPlace(newPlace *PlaceForClustering, existingPlaces []PlaceForClustering) bool {
	filtering := GetPlaceFiltering()
	minDistance := filtering.MinimumDistance / 1000.0 // Convert to kilometers
	
	for _, existing := range existingPlaces {
		distance := CalculateDistance(newPlace.Latitude, newPlace.Longitude, existing.Latitude, existing.Longitude)
		
		if distance < minDistance {
			// Aynı kategoride yakın yerler varsa cluster
			for _, newCat := range newPlace.Categories {
				for _, existingCat := range existing.Categories {
					if strings.ToLower(newCat) == strings.ToLower(existingCat) {
						// Daha kaliteli olanı seç
						if newPlace.Rating != nil && existing.Rating != nil {
							if *newPlace.Rating > *existing.Rating {
								return false // Yeni yer daha iyi, eskisini değiştir
							}
						}
						return true // Mevcut yer daha iyi veya eşit, yeniyi ekleme
					}
				}
			}
		}
	}
	
	return false
}

type PlaceForClustering struct {
	Latitude   float64
	Longitude  float64
	Categories []string
	Rating     *float64
	Name       string
}

func CalculatePlacePoints(categories []string, rating *float64, userRatingsTotal *int) int {
	scoring := GetPlaceScoring()
	basePoints := 15 // Varsayılan 15 puan (10-60 aralığında)
	maxCategoryPoints := 0
	
	// En yüksek kategori puanını bul
	for _, category := range categories {
		categoryLower := strings.ToLower(category)
		if points, exists := scoring.CategoryPoints[categoryLower]; exists && points > maxCategoryPoints {
			maxCategoryPoints = points
		}
	}
	
	if maxCategoryPoints > 0 {
		basePoints = maxCategoryPoints
	}
	
	// Rating bonusu
	ratingBonus := 0
	if rating != nil {
		switch {
		case *rating >= 4.5:
			ratingBonus = 15 // Mükemmel yerler için yüksek bonus
		case *rating >= 4.0:
			ratingBonus = 10
		case *rating >= 3.5:
			ratingBonus = 5
		case *rating >= 3.0:
			ratingBonus = 0
		default:
			ratingBonus = -10 // Düşük rating için ceza
		}
	}
	
	// Popülerlik bonusu - çok popüler yerleri önceliklendirme
	popularityBonus := 0
	if userRatingsTotal != nil {
		switch {
		case *userRatingsTotal >= 1000:
			popularityBonus = 20 // Çok popüler yerler
		case *userRatingsTotal >= 500:
			popularityBonus = 15
		case *userRatingsTotal >= 200:
			popularityBonus = 10
		case *userRatingsTotal >= 50:
			popularityBonus = 5
		case *userRatingsTotal >= 10:
			popularityBonus = 0
		default:
			popularityBonus = -5 // Az bilinen yerler için hafif ceza
		}
	}
	
	// Özel kategori kombinasyonları için bonus
	specialBonus := 0
	categorySet := make(map[string]bool)
	for _, cat := range categories {
		categorySet[strings.ToLower(cat)] = true
	}
	
	// UNESCO veya tarihi önem taşıyan yerler
	if categorySet["historical_site"] && categorySet["tourist_attraction"] {
		specialBonus += 15
	}
	
	// Doğal güzellik + turizm
	if categorySet["natural_feature"] && categorySet["tourist_attraction"] {
		specialBonus += 10
	}
	
	// Kültür + sanat
	if categorySet["museum"] && categorySet["art_gallery"] {
		specialBonus += 5
	}
	
	// Toplam puanı hesapla
	totalPoints := basePoints + ratingBonus + popularityBonus + specialBonus
	
	// 10-60 aralığına sınırla ve 5'in katları yap
	if totalPoints < 10 {
		totalPoints = 10
	} else if totalPoints > 60 {
		totalPoints = 60
	}
	
	// En yakın 5'in katına yuvarla
	finalPoints := ((totalPoints + 2) / 5) * 5
	
	return finalPoints
} 