package repository

import (
	"testing"
	"time"

	"homework/internal/model"
)

func TestMovieEntityToDomain(t *testing.T) {
	createdAt := time.Date(2026, 10, 12, 19, 0, 0, 123456000, time.UTC)
	year := 2020
	synopsis := "A synopsis"
	movie, err := movieEntityToDomain(model.MovieEntity{
		ID:          7,
		Title:       "Film",
		Country:     "Country",
		ReleaseYear: &year,
		Synopsis:    &synopsis,
		Genre:       model.Genre{Name: "Drama"},
		Directors: []model.Director{
			{ID: 9, FullName: "Second"},
			{ID: 3, FullName: "First"},
		},
		CreatedAt: createdAt,
		UpdatedAt: createdAt,
	})
	if err != nil {
		t.Fatalf("movieEntityToDomain() error = %v", err)
	}
	if movie.ID != 7 || movie.Director != "First" || movie.Genre != "Drama" {
		t.Fatalf("mapped identity/relations = (%d, %q, %q)", movie.ID, movie.Director, movie.Genre)
	}
	if movie.ReleaseYear == nil || *movie.ReleaseYear != year || movie.Synopsis == nil || *movie.Synopsis != synopsis {
		t.Fatalf("mapped optional fields = (%v, %v)", movie.ReleaseYear, movie.Synopsis)
	}
	if !movie.CreatedAt.Equal(createdAt) || !movie.UpdatedAt.Equal(createdAt) {
		t.Fatalf("mapped timestamps = (%s, %s)", movie.CreatedAt, movie.UpdatedAt)
	}
}

func TestMovieEntityToDomainRejectsIDOutsideAPIContract(t *testing.T) {
	if _, err := movieEntityToDomain(model.MovieEntity{ID: uint64(^uint32(0)) + 1}); err == nil {
		t.Fatal("expected uint32 range error")
	}
}

func TestMovieDomainToEntity(t *testing.T) {
	year := 2022
	synopsis := "Synopsis"
	createdAt := time.Date(2026, 10, 12, 19, 0, 0, 123456000, time.UTC)
	entity := movieDomainToEntity(model.Movie{
		ID:          12,
		Title:       "Film",
		Director:    "Director",
		Country:     "Country",
		Genre:       "Drama",
		ReleaseYear: &year,
		Synopsis:    &synopsis,
		CreatedAt:   createdAt,
		UpdatedAt:   createdAt,
	})
	if entity.ID != 12 || entity.Title != "Film" || entity.Country != "Country" {
		t.Fatalf("mapped entity = %#v", entity)
	}
	if entity.ReleaseYear == nil || *entity.ReleaseYear != year || entity.Synopsis == nil || *entity.Synopsis != synopsis {
		t.Fatalf("mapped optional fields = (%v, %v)", entity.ReleaseYear, entity.Synopsis)
	}
	if !entity.CreatedAt.Equal(createdAt) || !entity.UpdatedAt.Equal(createdAt) {
		t.Fatalf("mapped timestamps = (%s, %s)", entity.CreatedAt, entity.UpdatedAt)
	}
}
