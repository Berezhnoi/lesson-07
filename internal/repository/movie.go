package repository

import (
	"errors"
	"sort"
	"sync"

	"homework/internal/model"
)

type MovieRepository interface {
	Create(movie model.Movie) (model.Movie, error)
	GetByID(id uint32) (model.Movie, bool)
	Update(movie model.Movie) (model.Movie, bool)
	Delete(id uint32) bool
	Count(filter string) int
	List(filter string, offset, limit int) []model.Movie
}

type InMemoryMovieRepository struct {
	mu     sync.RWMutex
	items  map[uint32]model.Movie
	nextID uint64
}

// ErrIDExhausted indicates that all uint32 movie IDs have been allocated.
var ErrIDExhausted = errors.New("movie ID space exhausted")

var _ MovieRepository = (*InMemoryMovieRepository)(nil)

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
