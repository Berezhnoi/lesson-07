package repository

import (
	"context"
	"sort"
	"strings"

	"homework/internal/model"
)

var _ MovieSearcher = (*InMemoryMovieRepository)(nil)
var _ MovieSearcher = (*inMemoryMovieRepositoryAdapter)(nil)

func (repository *InMemoryMovieRepository) Search(ctx context.Context, q, genre string, offset, limit int) ([]model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	query := strings.ToLower(q)
	ids := make([]uint32, 0, len(repository.items))
	for id, movie := range repository.items {
		if genre != "" && movie.Genre != genre {
			continue
		}
		if query != "" &&
			!strings.Contains(strings.ToLower(movie.Title), query) &&
			!strings.Contains(strings.ToLower(movie.Director), query) {
			continue
		}
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })

	if offset < 0 || limit <= 0 || offset >= len(ids) {
		return make([]model.Movie, 0), nil
	}
	end := len(ids)
	if limit < len(ids)-offset {
		end = offset + limit
	}
	movies := make([]model.Movie, 0, end-offset)
	for _, id := range ids[offset:end] {
		movies = append(movies, cloneMovie(repository.items[id]))
	}
	return movies, nil
}

func (repository *InMemoryMovieRepository) Genres(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}

	repository.mu.RLock()
	defer repository.mu.RUnlock()

	genres := make([]string, 0)
	seen := make(map[string]struct{}, len(repository.items))
	for _, movie := range repository.items {
		if _, exists := seen[movie.Genre]; exists {
			continue
		}
		seen[movie.Genre] = struct{}{}
		genres = append(genres, movie.Genre)
	}
	sort.Strings(genres)
	return genres, nil
}

func (adapter *inMemoryMovieRepositoryAdapter) Genres(ctx context.Context) ([]string, error) {
	return adapter.repository.Genres(ctx)
}

func (adapter *inMemoryMovieRepositoryAdapter) Search(ctx context.Context, q, genre string, offset, limit int) ([]model.Movie, error) {
	return adapter.repository.Search(ctx, q, genre, offset, limit)
}
