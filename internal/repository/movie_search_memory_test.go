package repository

import (
	"context"
	"reflect"
	"testing"

	"homework/internal/model"
)

func TestInMemoryMovieSearch(t *testing.T) {
	ctx := context.Background()
	repository := NewInMemoryMovieRepositoryAdapter()
	for _, movie := range []model.Movie{
		{Title: "The Matrix", Director: "Lana Wachowski", Genre: "Sci-Fi"},
		{Title: "Matrix Reloaded", Director: "Lilly Wachowski", Genre: "Sci-Fi"},
		{Title: "A_B Story", Director: "Percent% Director", Genre: "Drama"},
		{Title: "Small Film", Director: "Another Director", Genre: "drama"},
	} {
		if _, err := repository.Create(ctx, movie); err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	searcher, ok := repository.(MovieSearcher)
	if !ok {
		t.Fatal("in-memory repository does not implement MovieSearcher")
	}

	tests := []struct {
		name   string
		q      string
		genre  string
		offset int
		limit  int
		want   []string
	}{
		{name: "case insensitive title match", q: "mAtRiX", limit: 10, want: []string{"The Matrix", "Matrix Reloaded"}},
		{name: "director match", q: "WACHOWSKI", limit: 10, want: []string{"The Matrix", "Matrix Reloaded"}},
		{name: "percent wildcard is literal", q: "%", limit: 10, want: []string{"A_B Story"}},
		{name: "underscore wildcard is literal", q: "_", limit: 10, want: []string{"A_B Story"}},
		{name: "genre exact match", genre: "Drama", limit: 10, want: []string{"A_B Story"}},
		{name: "genre is case sensitive", genre: "drama", limit: 10, want: []string{"Small Film"}},
		{name: "pagination keeps id order", q: "matrix", offset: 1, limit: 1, want: []string{"Matrix Reloaded"}},
		{name: "no matches returns empty slice", q: "absent", limit: 10, want: []string{}},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			movies, err := searcher.Search(ctx, test.q, test.genre, test.offset, test.limit)
			if err != nil {
				t.Fatalf("Search returned error: %v", err)
			}
			titles := make([]string, 0, len(movies))
			for _, movie := range movies {
				titles = append(titles, movie.Title)
			}
			if !reflect.DeepEqual(titles, test.want) {
				t.Errorf("titles = %#v, want %#v", titles, test.want)
			}
			if movies == nil {
				t.Error("Search returned nil slice")
			}
		})
	}

	genres, err := searcher.Genres(ctx)
	if err != nil {
		t.Fatalf("Genres returned error: %v", err)
	}
	if want := []string{"Drama", "Sci-Fi", "drama"}; !reflect.DeepEqual(genres, want) {
		t.Errorf("Genres = %#v, want %#v", genres, want)
	}
}
