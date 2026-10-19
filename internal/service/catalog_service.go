package service

import (
	"context"
	"strconv"
	"unicode/utf8"

	"homework/internal/model"
	"homework/internal/repository"
)

type SearchParams struct {
	Q     string
	Genre string
	Page  int
	Limit int
}

type CatalogService struct {
	movies   repository.MovieRepository
	searcher repository.MovieSearcher
}

func NewCatalogService(searcher repository.MovieSearcher) *CatalogService {
	service := &CatalogService{searcher: searcher}
	if movies, ok := searcher.(repository.MovieRepository); ok {
		service.movies = movies
	}
	return service
}

func ParseCatalogPagination(pageValue, limitValue string, hasPage, hasLimit bool) (int, int, error) {
	page, limit, err := parsePaginationValues(pageValue, limitValue, hasPage, hasLimit)
	if err != nil {
		return 0, 0, err
	}
	maxInt := uint64(^uint(0) >> 1)
	if page > maxInt {
		page = maxInt
	}
	return int(page), limit, nil
}

func (service *CatalogService) Search(ctx context.Context, params SearchParams) ([]model.Movie, bool, error) {
	if utf8.RuneCountInString(params.Q) > 100 {
		return nil, false, ValidationError{Message: "q must be at most 100 characters"}
	}
	page, limit, err := parsePaginationValues(strconv.Itoa(params.Page), strconv.Itoa(params.Limit), true, true)
	if err != nil {
		return nil, false, err
	}
	offset, empty := paginationOffset(page, limit, int(^uint(0)>>1))
	if empty {
		return make([]model.Movie, 0), false, nil
	}
	movies, err := service.searcher.Search(ctx, params.Q, params.Genre, offset, limit+1)
	if err != nil {
		return nil, false, repositoryError(err)
	}
	hasNext := len(movies) > limit
	if hasNext {
		movies = movies[:limit]
	}
	if movies == nil {
		movies = make([]model.Movie, 0)
	}
	return movies, hasNext, nil
}

func (service *CatalogService) Genres(ctx context.Context) ([]string, error) {
	return service.searcher.Genres(ctx)
}
