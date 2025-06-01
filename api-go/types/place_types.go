package types

type NearbyPlacesRequest struct {
	Latitude       float64 `form:"latitude" binding:"required"`
	Longitude      float64 `form:"longitude" binding:"required"`
	ZoomLevel      int     `form:"zoomLevel" binding:"required,min=1,max=20"`
	Radius         float64 `form:"radius"`
	HideVisited    bool    `form:"hideVisited"`
	CategoryFilter string  `form:"category"`
	MaxPlaces      int     `form:"maxPlaces"`
}

type Marker struct {
	ID         uint    `json:"id" gorm:"column:id"`
	Latitude   float64 `json:"latitude" gorm:"column:latitude"`
	Longitude  float64 `json:"longitude" gorm:"column:longitude"`
	PointValue int     `json:"pointValue" gorm:"column:point_value"`
	IsVerified bool    `json:"isVerified" gorm:"column:is_verified"`
}

type NearbyPlacesResponse struct {
	Markers []Marker `json:"markers"`
	Filters struct {
		Radius      float64 `json:"radius"`
		ZoomLevel   int     `json:"zoomLevel"`
		HideVisited bool    `json:"hideVisited"`
		Category    string  `json:"category"`
	} `json:"filters"`
}
