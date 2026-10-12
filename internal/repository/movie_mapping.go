package repository

import (
	"fmt"
	"sort"

	"homework/internal/model"
)

func movieEntityToDomain(entity model.MovieEntity) (model.Movie, error) {
	if entity.ID > uint64(^uint32(0)) {
		return model.Movie{}, fmt.Errorf("movie ID %d exceeds API uint32 range", entity.ID)
	}
	directors := append([]model.Director(nil), entity.Directors...)
	sort.Slice(directors, func(i, j int) bool { return directors[i].ID < directors[j].ID })
	director := ""
	if len(directors) != 0 {
		director = directors[0].FullName
	}
	return model.Movie{
		ID:          uint32(entity.ID),
		Title:       entity.Title,
		Director:    director,
		Country:     entity.Country,
		Genre:       entity.Genre.Name,
		ReleaseYear: entity.ReleaseYear,
		Synopsis:    entity.Synopsis,
		CreatedAt:   entity.CreatedAt,
		UpdatedAt:   entity.UpdatedAt,
	}, nil
}

func movieDomainToEntity(movie model.Movie) model.MovieEntity {
	return model.MovieEntity{
		ID:          uint64(movie.ID),
		Title:       movie.Title,
		Country:     movie.Country,
		ReleaseYear: movie.ReleaseYear,
		Synopsis:    movie.Synopsis,
		CreatedAt:   movie.CreatedAt,
		UpdatedAt:   movie.UpdatedAt,
	}
}
