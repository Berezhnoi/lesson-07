package repository

import (
	"context"
	"strings"

	"gorm.io/gorm"

	"homework/internal/model"
)

var _ MovieSearcher = (*gormMovieRepository)(nil)

func (repository *gormMovieRepository) Search(ctx context.Context, q, genre string, offset, limit int) ([]model.Movie, error) {
	entities := make([]model.MovieEntity, 0)
	query := repository.db.WithContext(ctx).
		Model(&model.MovieEntity{}).
		Preload("Genre").
		Preload("Directors", func(db *gorm.DB) *gorm.DB { return db.Order("directors.id ASC") }).
		Order("movies.id ASC").
		Offset(offset).
		Limit(limit)
	if genre != "" {
		query = query.Where(
			"movies.genre_id IN (SELECT genres.id FROM genres WHERE genres.name = ?)",
			genre,
		)
	}
	if q != "" {
		pattern := "%" + escapeMovieSearchLike(q) + "%"
		query = query.Where(
			`(movies.title ILIKE ? ESCAPE '\' OR EXISTS (
				SELECT 1
				FROM movie_directors AS md
				JOIN directors AS d ON d.id = md.director_id
				WHERE md.movie_id = movies.id AND d.full_name ILIKE ? ESCAPE '\'
			))`,
			pattern,
			pattern,
		)
	}
	if err := query.Find(&entities).Error; err != nil {
		return nil, wrapDatabaseError("search movies", err)
	}
	movies := make([]model.Movie, 0, len(entities))
	for _, entity := range entities {
		movie, err := movieEntityToDomain(entity)
		if err != nil {
			return nil, err
		}
		movies = append(movies, movie)
	}
	return movies, nil
}

func (repository *gormMovieRepository) Genres(ctx context.Context) ([]string, error) {
	genres := make([]string, 0)
	err := repository.db.WithContext(ctx).
		Table("genres").
		Select("DISTINCT genres.name").
		Joins("JOIN movies ON movies.genre_id = genres.id").
		Order("genres.name ASC").
		Pluck("genres.name", &genres).Error
	if err != nil {
		return nil, wrapDatabaseError("list movie genres", err)
	}
	return genres, nil
}

func escapeMovieSearchLike(value string) string {
	return strings.NewReplacer(`\`, `\\`, `%`, `\%`, `_`, `\_`).Replace(value)
}
