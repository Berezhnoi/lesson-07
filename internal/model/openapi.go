package model

// MovieRequest documents the JSON body accepted by POST and PUT.
// Runtime validation is performed by the service, not Gin binding.
type MovieRequest struct {
	Title    string `json:"title" binding:"required" example:"The Matrix"`
	Director string `json:"director" binding:"required" example:"Lana Wachowski"`
	Country  string `json:"country" binding:"required" example:"United States"`
	Genre    string `json:"genre" binding:"required" example:"Science Fiction"`
	// ReleaseYear may be omitted or set to null.
	ReleaseYear *int `json:"release_year" example:"1999"`
	// Synopsis may be omitted or set to null; an empty string is valid.
	Synopsis *string `json:"synopsis" example:"A computer hacker learns about reality."`
}

// MovieResponse documents a serialized movie resource.
type MovieResponse struct {
	ID       uint32 `json:"id" example:"1" minimum:"1" maximum:"4294967295"`
	Title    string `json:"title" example:"The Matrix"`
	Director string `json:"director" example:"Lana Wachowski"`
	Country  string `json:"country" example:"United States"`
	Genre    string `json:"genre" example:"Science Fiction"`
	// ReleaseYear is always present and may be null.
	ReleaseYear *int `json:"release_year" example:"1999"`
	// Synopsis is always present and may be null.
	Synopsis  *string `json:"synopsis" example:"A computer hacker learns about reality."`
	CreatedAt string  `json:"created_at" format:"date-time" example:"2026-10-05T12:00:00.123456Z"`
	UpdatedAt string  `json:"updated_at" format:"date-time" example:"2026-10-05T12:00:00.123456Z"`
}

// ErrorResponse documents the API error envelope.
type ErrorResponse struct {
	Error ErrorDetail `json:"error"`
}

// ErrorDetail contains a stable error code and human-readable message.
type ErrorDetail struct {
	Code    string `json:"code" example:"validation_error"`
	Message string `json:"message" example:"title is required"`
}

// HealthResponse is the response from GET /health.
type HealthResponse struct {
	Status string `json:"status" example:"ok"`
}
