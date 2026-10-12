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

type Genre struct {
	ID   uint64 `gorm:"primaryKey"`
	Name string `gorm:"uniqueIndex:uq_genres_name;not null"`
}

func (Genre) TableName() string { return "genres" }

type Director struct {
	ID       uint64 `gorm:"primaryKey"`
	FullName string `gorm:"column:full_name;uniqueIndex:uq_directors_full_name;not null"`
	Bio      *string
}

func (Director) TableName() string { return "directors" }

type MovieEntity struct {
	ID          uint64 `gorm:"primaryKey"`
	Title       string `gorm:"not null"`
	Country     string `gorm:"not null"`
	ReleaseYear *int   `gorm:"column:release_year"`
	Synopsis    *string
	GenreID     uint64     `gorm:"not null"`
	Genre       Genre      `gorm:"foreignKey:GenreID"`
	Directors   []Director `gorm:"many2many:movie_directors;foreignKey:ID;joinForeignKey:MovieID;references:ID;joinReferences:DirectorID"`
	CreatedAt   time.Time  `gorm:"autoCreateTime:false"`
	UpdatedAt   time.Time  `gorm:"autoUpdateTime:false"`
}

func (MovieEntity) TableName() string { return "movies" }

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
