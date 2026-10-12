package repository

import (
	"context"
	"errors"
	"sort"
	"sync"

	"homework/internal/model"
)

type MovieRepository interface {
	Create(ctx context.Context, movie model.Movie) (model.Movie, error)
	GetByID(ctx context.Context, id uint32) (model.Movie, bool)
	Update(ctx context.Context, movie model.Movie) (model.Movie, bool)
	Delete(ctx context.Context, id uint32) bool
	Count(ctx context.Context, filter string) int
	List(ctx context.Context, filter string, offset, limit int) []model.Movie
}

// MovieRepositoryWithErrors preserves backend failures without changing the legacy result shapes.
type MovieRepositoryWithErrors interface {
	GetByIDWithError(ctx context.Context, id uint32) (model.Movie, error)
	UpdateWithError(ctx context.Context, movie model.Movie) (model.Movie, error)
	DeleteWithError(ctx context.Context, id uint32) error
	CountWithError(ctx context.Context, filter string) (int, error)
	ListWithError(ctx context.Context, filter string, offset, limit int) ([]model.Movie, error)
}

var ErrNotFound = errors.New("movie not found")

type InMemoryMovieRepository struct {
	mu     sync.RWMutex
	items  map[uint32]model.Movie
	nextID uint64
}

// ErrIDExhausted indicates that all uint32 movie IDs have been allocated.
var ErrIDExhausted = errors.New("movie ID space exhausted")

func NewInMemoryMovieRepository() *InMemoryMovieRepository {
	return &InMemoryMovieRepository{items: make(map[uint32]model.Movie), nextID: 1}
}

func (repository *InMemoryMovieRepository) Create(movie model.Movie) (model.Movie, error) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if repository.nextID > uint64(^uint32(0)) {
		return model.Movie{}, ErrIDExhausted
	}
	movie.ID = uint32(repository.nextID)
	repository.nextID++
	repository.items[movie.ID] = cloneMovie(movie)
	return cloneMovie(movie), nil
}

func (repository *InMemoryMovieRepository) GetByID(id uint32) (model.Movie, bool) {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	movie, ok := repository.items[id]
	return cloneMovie(movie), ok
}

func (repository *InMemoryMovieRepository) Update(movie model.Movie) (model.Movie, bool) {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, ok := repository.items[movie.ID]; !ok {
		return model.Movie{}, false
	}
	repository.items[movie.ID] = cloneMovie(movie)
	return cloneMovie(movie), true
}

func (repository *InMemoryMovieRepository) Delete(id uint32) bool {
	repository.mu.Lock()
	defer repository.mu.Unlock()

	if _, ok := repository.items[id]; !ok {
		return false
	}
	delete(repository.items, id)
	return true
}

func (repository *InMemoryMovieRepository) Count(filter string) int {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	count := 0
	for _, movie := range repository.items {
		if filter == "" || movie.Genre == filter {
			count++
		}
	}
	return count
}

func (repository *InMemoryMovieRepository) List(filter string, offset, limit int) []model.Movie {
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	ids := make([]uint32, 0, len(repository.items))
	for id, movie := range repository.items {
		if filter == "" || movie.Genre == filter {
			ids = append(ids, id)
		}
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if offset < 0 || limit <= 0 || offset >= len(ids) {
		return []model.Movie{}
	}
	end := len(ids)
	if limit < len(ids)-offset {
		end = offset + limit
	}

	result := make([]model.Movie, 0, end-offset)
	for _, id := range ids[offset:end] {
		result = append(result, cloneMovie(repository.items[id]))
	}
	return result
}

type inMemoryMovieRepositoryAdapter struct {
	repository *InMemoryMovieRepository
}

var _ MovieRepository = (*inMemoryMovieRepositoryAdapter)(nil)
var _ MovieRepositoryWithErrors = (*inMemoryMovieRepositoryAdapter)(nil)

func NewInMemoryMovieRepositoryAdapter() MovieRepository {
	return &inMemoryMovieRepositoryAdapter{repository: NewInMemoryMovieRepository()}
}

func (adapter *inMemoryMovieRepositoryAdapter) Create(ctx context.Context, movie model.Movie) (model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return model.Movie{}, err
	}
	return adapter.repository.Create(movie)
}

func (adapter *inMemoryMovieRepositoryAdapter) GetByID(ctx context.Context, id uint32) (model.Movie, bool) {
	movie, err := adapter.GetByIDWithError(ctx, id)
	return movie, err == nil
}

func (adapter *inMemoryMovieRepositoryAdapter) GetByIDWithError(ctx context.Context, id uint32) (model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return model.Movie{}, err
	}
	movie, ok := adapter.repository.GetByID(id)
	if !ok {
		return model.Movie{}, ErrNotFound
	}
	return movie, nil
}

func (adapter *inMemoryMovieRepositoryAdapter) Update(ctx context.Context, movie model.Movie) (model.Movie, bool) {
	updated, err := adapter.UpdateWithError(ctx, movie)
	return updated, err == nil
}

func (adapter *inMemoryMovieRepositoryAdapter) UpdateWithError(ctx context.Context, movie model.Movie) (model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return model.Movie{}, err
	}
	updated, ok := adapter.repository.Update(movie)
	if !ok {
		return model.Movie{}, ErrNotFound
	}
	return updated, nil
}

func (adapter *inMemoryMovieRepositoryAdapter) Delete(ctx context.Context, id uint32) bool {
	return adapter.DeleteWithError(ctx, id) == nil
}

func (adapter *inMemoryMovieRepositoryAdapter) DeleteWithError(ctx context.Context, id uint32) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if !adapter.repository.Delete(id) {
		return ErrNotFound
	}
	return nil
}

func (adapter *inMemoryMovieRepositoryAdapter) Count(ctx context.Context, filter string) int {
	count, _ := adapter.CountWithError(ctx, filter)
	return count
}

func (adapter *inMemoryMovieRepositoryAdapter) CountWithError(ctx context.Context, filter string) (int, error) {
	if err := ctx.Err(); err != nil {
		return 0, err
	}
	return adapter.repository.Count(filter), nil
}

func (adapter *inMemoryMovieRepositoryAdapter) List(ctx context.Context, filter string, offset, limit int) []model.Movie {
	movies, _ := adapter.ListWithError(ctx, filter, offset, limit)
	return movies
}

func (adapter *inMemoryMovieRepositoryAdapter) ListWithError(ctx context.Context, filter string, offset, limit int) ([]model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	return adapter.repository.List(filter, offset, limit), nil
}

func cloneMovie(movie model.Movie) model.Movie {
	if movie.ReleaseYear != nil {
		year := *movie.ReleaseYear
		movie.ReleaseYear = &year
	}
	if movie.Synopsis != nil {
		synopsis := *movie.Synopsis
		movie.Synopsis = &synopsis
	}
	return movie
}
