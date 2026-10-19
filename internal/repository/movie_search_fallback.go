package repository

import (
	"context"
	"sort"
	"strings"

	"homework/internal/model"
)

type repositoryMovieSearcher struct {
	movies MovieRepository
}

func NewMovieSearcher(movies MovieRepository) MovieSearcher {
	if searcher, ok := movies.(MovieSearcher); ok {
		return searcher
	}
	return repositoryMovieSearcher{movies: movies}
}

func (searcher repositoryMovieSearcher) Search(ctx context.Context, query, genre string, offset, limit int) ([]model.Movie, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	total, err := searcher.count(ctx, genre)
	if err != nil {
		return nil, err
	}
	movies, err := searcher.list(ctx, genre, 0, total)
	if err != nil {
		return nil, err
	}
	query = strings.ToLower(query)
	matched := make([]model.Movie, 0, len(movies))
	for _, movie := range movies {
		if query == "" || strings.Contains(strings.ToLower(movie.Title), query) ||
			strings.Contains(strings.ToLower(movie.Director), query) {
			matched = append(matched, movie)
		}
	}
	if offset < 0 || limit <= 0 || offset >= len(matched) {
		return []model.Movie{}, nil
	}
	end := offset + limit
	if end > len(matched) {
		end = len(matched)
	}
	return matched[offset:end], nil
}

func (searcher repositoryMovieSearcher) Genres(ctx context.Context) ([]string, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	total, err := searcher.count(ctx, "")
	if err != nil {
		return nil, err
	}
	movies, err := searcher.list(ctx, "", 0, total)
	if err != nil {
		return nil, err
	}
	seen := make(map[string]struct{}, len(movies))
	genres := make([]string, 0)
	for _, movie := range movies {
		if _, exists := seen[movie.Genre]; exists {
			continue
		}
		seen[movie.Genre] = struct{}{}
		genres = append(genres, movie.Genre)
	}
	sort.Strings(genres)
	return genres, nil
}

func (searcher repositoryMovieSearcher) count(ctx context.Context, genre string) (int, error) {
	if repository, ok := searcher.movies.(MovieRepositoryWithErrors); ok {
		return repository.CountWithError(ctx, genre)
	}
	return searcher.movies.Count(ctx, genre), nil
}

func (searcher repositoryMovieSearcher) list(ctx context.Context, genre string, offset, limit int) ([]model.Movie, error) {
	if repository, ok := searcher.movies.(MovieRepositoryWithErrors); ok {
		return repository.ListWithError(ctx, genre, offset, limit)
	}
	return searcher.movies.List(ctx, genre, offset, limit), nil
}
