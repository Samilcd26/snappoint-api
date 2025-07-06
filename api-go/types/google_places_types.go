package types

type GooglePlacesResponse struct {
	HTMLAttributions []string             `json:"html_attributions"`
	NextPageToken    string               `json:"next_page_token"`
	Results          []GooglePlaceResult  `json:"results"`
	Status           string               `json:"status"`
}

// Alias for backward compatibility and clarity
type GooglePlaceResult = PlaceResult

type PlaceResult struct {
	BusinessStatus      *string       `json:"business_status,omitempty"`
	Geometry            Geometry      `json:"geometry"`
	Icon                string        `json:"icon"`
	IconBackgroundColor string        `json:"icon_background_color"`
	IconMaskBaseURI     string        `json:"icon_mask_base_uri"`
	Name                string        `json:"name"`
	OpeningHours        *OpeningHours `json:"opening_hours,omitempty"`
	Photos              []Photo       `json:"photos,omitempty"`
	PlaceID             string        `json:"place_id"`
	PlusCode            *PlusCode     `json:"plus_code,omitempty"`
	Rating              *float64      `json:"rating,omitempty"`
	Reference           string        `json:"reference"`
	Scope               string        `json:"scope"`
	Types               []string      `json:"types"`
	UserRatingsTotal    *int          `json:"user_ratings_total,omitempty"`
	Vicinity            *string       `json:"vicinity,omitempty"`
}

type Geometry struct {
	Location Location `json:"location"`
	Viewport Viewport `json:"viewport"`
}

type Location struct {
	Lat float64 `json:"lat"`
	Lng float64 `json:"lng"`
}

type Viewport struct {
	Northeast Location `json:"northeast"`
	Southwest Location `json:"southwest"`
}

type OpeningHours struct {
	OpenNow bool `json:"open_now"`
}

type Photo struct {
	Height           int      `json:"height"`
	HTMLAttributions []string `json:"html_attributions"`
	PhotoReference   string   `json:"photo_reference"`
	Width            int      `json:"width"`
}

type PlusCode struct {
	CompoundCode string `json:"compound_code"`
	GlobalCode   string `json:"global_code"`
} 