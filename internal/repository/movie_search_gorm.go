package repository

import (
	"context"
	"strings"
	"time"

	"homework/internal/model"
)

var _ MovieSearcher = (*gormMovieRepository)(nil)

func (repository *gormMovieRepository) Search(ctx context.Context, q, genre string, offset, limit int) ([]model.Movie, error) {
	type searchRow struct {
		ID          uint64
		Title       string
		Director    string `gorm:"column:director_name"`
		Country     string
		Genre       string `gorm:"column:genre_name"`
		ReleaseYear *int
		Synopsis    *string
		CreatedAt   time.Time
		UpdatedAt   time.Time
	}

	rows := make([]searchRow, 0)
	query := repository.db.WithContext(ctx).
		Table("movies").
		Select(`movies.id, movies.title, movies.country, movies.release_year, movies.synopsis,
			movies.created_at, movies.updated_at, genres.name AS genre_name,
			(SELECT directors.full_name
			 FROM movie_directors
			 JOIN directors ON directors.id = movie_directors.director_id
			 WHERE movie_directors.movie_id = movies.id
			 ORDER BY directors.id ASC
			 LIMIT 1) AS director_name`).
		Joins("JOIN genres ON genres.id = movies.genre_id").
		Order("movies.id ASC").
		Offset(offset).
		Limit(limit)
	if genre != "" {
		query = query.Where("genres.name = ?", genre)
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
	if err := query.Scan(&rows).Error; err != nil {
		return nil, wrapDatabaseError("search movies", err)
	}
	movies := make([]model.Movie, 0, len(rows))
	for _, row := range rows {
		movie, err := movieEntityToDomain(model.MovieEntity{
			ID:          row.ID,
			Title:       row.Title,
			Country:     row.Country,
			ReleaseYear: row.ReleaseYear,
			Synopsis:    row.Synopsis,
			Genre:       model.Genre{Name: row.Genre},
			Directors:   []model.Director{{FullName: row.Director}},
			CreatedAt:   row.CreatedAt,
			UpdatedAt:   row.UpdatedAt,
		})
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
