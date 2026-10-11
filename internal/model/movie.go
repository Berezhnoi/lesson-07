package model

import (
	"encoding/json"
	"time"
)

type Movie struct {
	ID          uint32    `json:"id"`
	Title       string    `json:"title"`
	Director    string    `json:"director"`
	Country     string    `json:"country"`
	Genre       string    `json:"genre"`
	ReleaseYear *int      `json:"release_year"`
	Synopsis    *string   `json:"synopsis"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type MovieInput struct {
	Title       string  `json:"title"`
	Director    string  `json:"director"`
	Country     string  `json:"country"`
	Genre       string  `json:"genre"`
	ReleaseYear *int    `json:"release_year"`
	Synopsis    *string `json:"synopsis"`
}

func (movie Movie) MarshalJSON() ([]byte, error) {
	type movieResponse struct {
		ID          uint32  `json:"id"`
		Title       string  `json:"title"`
		Director    string  `json:"director"`
		Country     string  `json:"country"`
		Genre       string  `json:"genre"`
		ReleaseYear *int    `json:"release_year"`
		Synopsis    *string `json:"synopsis"`
		CreatedAt   string  `json:"created_at"`
		UpdatedAt   string  `json:"updated_at"`
	}

	return json.Marshal(movieResponse{
		ID:          movie.ID,
		Title:       movie.Title,
		Director:    movie.Director,
		Country:     movie.Country,
		Genre:       movie.Genre,
		ReleaseYear: movie.ReleaseYear,
		Synopsis:    movie.Synopsis,
		CreatedAt:   movie.CreatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
		UpdatedAt:   movie.UpdatedAt.UTC().Format("2006-01-02T15:04:05.000000Z"),
	})
}
