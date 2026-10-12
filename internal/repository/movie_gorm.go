package repository

import (
	"context"
	"errors"
	"fmt"
	"log"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"homework/internal/model"
)

type gormMovieRepository struct {
	db *gorm.DB
}

var _ MovieRepository = (*gormMovieRepository)(nil)
var _ MovieRepositoryWithErrors = (*gormMovieRepository)(nil)

func NewGormMovieRepository(db *gorm.DB) MovieRepository {
	return &gormMovieRepository{db: db}
}

func (repository *gormMovieRepository) Create(ctx context.Context, movie model.Movie) (model.Movie, error) {
	var entity model.MovieEntity
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		genre, err := findOrCreateGenre(tx, movie.Genre)
		if err != nil {
			return err
		}
		director, err := findOrCreateDirector(tx, movie.Director)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Truncate(time.Microsecond)
		entity = movieDomainToEntity(movie)
		entity.GenreID = genre.ID
		entity.CreatedAt = now
		entity.UpdatedAt = now
		if err := tx.Omit("Genre", "Directors").Create(&entity).Error; err != nil {
			return err
		}
		if entity.ID > uint64(^uint32(0)) {
			return fmt.Errorf("movie ID %d exceeds API uint32 range", entity.ID)
		}
		if err := tx.Model(&entity).Association("Directors").Append(&director); err != nil {
			return err
		}
		entity.Genre = genre
		entity.Directors = []model.Director{director}
		return nil
	})
	if err != nil {
		return model.Movie{}, wrapDatabaseError("create movie", err)
	}
	return movieEntityToDomain(entity)
}

func (repository *gormMovieRepository) GetByID(ctx context.Context, id uint32) (model.Movie, bool) {
	movie, err := repository.GetByIDWithError(ctx, id)
	if err != nil {
		log.Printf("get movie failed: %v", err)
		return model.Movie{}, false
	}
	return movie, true
}

func (repository *gormMovieRepository) GetByIDWithError(ctx context.Context, id uint32) (model.Movie, error) {
	var entity model.MovieEntity
	err := repository.db.WithContext(ctx).
		Preload("Genre").
		Preload("Directors", func(db *gorm.DB) *gorm.DB { return db.Order("directors.id ASC") }).
		First(&entity, "movies.id = ?", id).Error
	if err != nil {
		return model.Movie{}, wrapDatabaseError("find movie", err)
	}
	return movieEntityToDomain(entity)
}

func (repository *gormMovieRepository) Update(ctx context.Context, movie model.Movie) (model.Movie, bool) {
	updated, err := repository.UpdateWithError(ctx, movie)
	if err != nil {
		log.Printf("update movie failed: %v", err)
		return model.Movie{}, false
	}
	return updated, true
}

func (repository *gormMovieRepository) UpdateWithError(ctx context.Context, movie model.Movie) (model.Movie, error) {
	var saved model.MovieEntity
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.MovieEntity
		if err := tx.Preload("Genre").Preload("Directors").First(&current, "movies.id = ?", movie.ID).Error; err != nil {
			return err
		}
		genre, err := findOrCreateGenre(tx, movie.Genre)
		if err != nil {
			return err
		}
		director, err := findOrCreateDirector(tx, movie.Director)
		if err != nil {
			return err
		}

		now := time.Now().UTC().Truncate(time.Microsecond)
		lastTimestamp := current.UpdatedAt
		if current.CreatedAt.After(lastTimestamp) {
			lastTimestamp = current.CreatedAt
		}
		if !now.After(lastTimestamp) {
			now = lastTimestamp.Add(time.Microsecond)
		}
		updates := map[string]any{
			"title":        movie.Title,
			"country":      movie.Country,
			"release_year": movie.ReleaseYear,
			"synopsis":     movie.Synopsis,
			"genre_id":     genre.ID,
			"updated_at":   now,
		}
		result := tx.Model(&model.MovieEntity{}).Where("id = ?", movie.ID).Updates(updates)
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}

		association := tx.Model(&current).Association("Directors")
		if err := association.Replace(&director); err != nil {
			return err
		}
		if err := tx.Preload("Genre").Preload("Directors", func(db *gorm.DB) *gorm.DB {
			return db.Order("directors.id ASC")
		}).First(&saved, "movies.id = ?", movie.ID).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return model.Movie{}, wrapDatabaseError("update movie", err)
	}
	return movieEntityToDomain(saved)
}

func (repository *gormMovieRepository) Delete(ctx context.Context, id uint32) bool {
	err := repository.DeleteWithError(ctx, id)
	if err != nil {
		if !errors.Is(err, ErrNotFound) {
			log.Printf("delete movie failed: %v", err)
		}
		return false
	}
	return true
}

func (repository *gormMovieRepository) DeleteWithError(ctx context.Context, id uint32) error {
	result := repository.db.WithContext(ctx).Delete(&model.MovieEntity{}, id)
	if result.Error != nil {
		return wrapDatabaseError("delete movie", result.Error)
	}
	if result.RowsAffected == 0 {
		return ErrNotFound
	}
	// PostgreSQL's BIGSERIAL sequence is not decremented by DELETE, so IDs are not reused.
	return nil
}

func (repository *gormMovieRepository) Count(ctx context.Context, filter string) int {
	count, err := repository.CountWithError(ctx, filter)
	if err != nil {
		log.Printf("count movies failed: %v", err)
		return 0
	}
	return count
}

func (repository *gormMovieRepository) CountWithError(ctx context.Context, filter string) (int, error) {
	query := repository.db.WithContext(ctx).Model(&model.MovieEntity{})
	if filter != "" {
		query = query.Joins("JOIN genres ON genres.id = movies.genre_id").Where("genres.name = ?", filter)
	}
	var count int64
	if err := query.Count(&count).Error; err != nil {
		return 0, wrapDatabaseError("count movies", err)
	}
	return int(count), nil
}

func (repository *gormMovieRepository) List(ctx context.Context, filter string, offset, limit int) []model.Movie {
	movies, err := repository.ListWithError(ctx, filter, offset, limit)
	if err != nil {
		log.Printf("list movies failed: %v", err)
		return nil
	}
	return movies
}

func (repository *gormMovieRepository) ListWithError(ctx context.Context, filter string, offset, limit int) ([]model.Movie, error) {
	entities := make([]model.MovieEntity, 0)
	query := repository.db.WithContext(ctx).
		Model(&model.MovieEntity{}).
		Joins("Genre").
		Preload("Directors", func(db *gorm.DB) *gorm.DB { return db.Order("directors.id ASC") }).
		Order("movies.id ASC").
		Offset(offset).
		Limit(limit)
	if filter != "" {
		query = query.Where(`"Genre"."name" = ?`, filter)
	}
	if err := query.Find(&entities).Error; err != nil {
		return nil, wrapDatabaseError("list movies", err)
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

func findOrCreateGenre(tx *gorm.DB, name string) (model.Genre, error) {
	genre := model.Genre{Name: name}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "name"}},
		DoNothing: true,
	}).Create(&genre).Error; err != nil {
		return model.Genre{}, err
	}
	if err := tx.Where("name = ?", name).First(&genre).Error; err != nil {
		return model.Genre{}, err
	}
	return genre, nil
}

func findOrCreateDirector(tx *gorm.DB, fullName string) (model.Director, error) {
	director := model.Director{FullName: fullName}
	if err := tx.Clauses(clause.OnConflict{
		Columns:   []clause.Column{{Name: "full_name"}},
		DoNothing: true,
	}).Create(&director).Error; err != nil {
		return model.Director{}, err
	}
	if err := tx.Where("full_name = ?", fullName).First(&director).Error; err != nil {
		return model.Director{}, err
	}
	return director, nil
}

func wrapDatabaseError(operation string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	return fmt.Errorf("%s: %w", operation, err)
}
