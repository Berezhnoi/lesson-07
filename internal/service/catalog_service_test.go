package service

import (
	"context"
	"errors"
	"reflect"
	"strings"
	"testing"

	"homework/internal/model"
	"homework/internal/repository"
)

func TestCatalogServiceSearchPagination(t *testing.T) {
	ctx := context.Background()
	movies := repository.NewInMemoryMovieRepositoryAdapter()
	for _, title := range []string{"One", "Two", "Three"} {
		if _, err := movies.Create(ctx, model.Movie{Title: title, Director: "Director", Genre: "Drama"}); err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	searcher, ok := movies.(repository.MovieSearcher)
	if !ok {
		t.Fatal("in-memory repository does not implement MovieSearcher")
	}
	catalog := NewCatalogService(searcher)

	first, hasNext, err := catalog.Search(ctx, SearchParams{Page: 1, Limit: 2})
	if err != nil {
		t.Fatalf("Search first page: %v", err)
	}
	if got := movieTitles(first); !reflect.DeepEqual(got, []string{"One", "Two"}) || !hasNext {
		t.Fatalf("first page = %#v, hasNext=%t; want [One Two], true", got, hasNext)
	}
	last, hasNext, err := catalog.Search(ctx, SearchParams{Page: 2, Limit: 2})
	if err != nil {
		t.Fatalf("Search last page: %v", err)
	}
	if got := movieTitles(last); !reflect.DeepEqual(got, []string{"Three"}) || hasNext {
		t.Fatalf("last page = %#v, hasNext=%t; want [Three], false", got, hasNext)
	}
	empty, hasNext, err := catalog.Search(ctx, SearchParams{Page: 1, Limit: 10, Q: "missing"})
	if err != nil {
		t.Fatalf("Search empty page: %v", err)
	}
	if empty == nil || len(empty) != 0 || hasNext {
		t.Fatalf("empty search = %#v, hasNext=%t; want non-nil empty, false", empty, hasNext)
	}
	largePage, hasNext, err := catalog.Search(ctx, SearchParams{Page: int(^uint(0) >> 1), Limit: 10})
	if err != nil {
		t.Fatalf("Search with maximum page: %v", err)
	}
	if largePage == nil || len(largePage) != 0 || hasNext {
		t.Fatalf("maximum-page search = %#v, hasNext=%t; want non-nil empty, false", largePage, hasNext)
	}
}

func TestCatalogServiceSearchValidationAndLimitCap(t *testing.T) {
	catalog := NewCatalogService(repository.NewInMemoryMovieRepositoryAdapter().(repository.MovieSearcher))
	for _, params := range []SearchParams{{Page: 0, Limit: 10}, {Page: 1, Limit: 0}, {Page: -1, Limit: 10}, {Page: 1, Limit: -1}} {
		_, _, err := catalog.Search(context.Background(), params)
		var invalid InvalidPaginationError
		if !errors.As(err, &invalid) {
			t.Errorf("Search(%+v) error = %T (%v), want InvalidPaginationError", params, err, err)
		}
	}
	_, _, err := catalog.Search(context.Background(), SearchParams{Q: strings.Repeat("x", 101), Page: 1, Limit: 10})
	var validation ValidationError
	if !errors.As(err, &validation) {
		t.Fatalf("long query error = %T (%v), want ValidationError", err, err)
	}

	for index := 0; index < 101; index++ {
		if _, err := catalog.searcher.(repository.MovieRepository).Create(context.Background(), model.Movie{
			Title: "Title", Director: "Director", Genre: "Drama",
		}); err != nil {
			t.Fatalf("create fixture %d: %v", index, err)
		}
	}
	movies, hasNext, err := catalog.Search(context.Background(), SearchParams{Page: 1, Limit: 101})
	if err != nil {
		t.Fatalf("Search capped limit: %v", err)
	}
	if len(movies) != 100 || !hasNext {
		t.Fatalf("capped result len=%d hasNext=%t, want 100 true", len(movies), hasNext)
	}
}

func movieTitles(movies []model.Movie) []string {
	titles := make([]string, 0, len(movies))
	for _, movie := range movies {
		titles = append(titles, movie.Title)
	}
	return titles
}
