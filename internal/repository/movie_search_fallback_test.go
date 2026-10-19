package repository

import (
	"context"
	"reflect"
	"testing"

	"homework/internal/model"
)

type movieRepositoryWithoutSearcher struct {
	MovieRepository
}

func TestMovieSearcherFallsBackToMovieRepository(t *testing.T) {
	ctx := context.Background()
	base := NewInMemoryMovieRepositoryAdapter()
	repository := movieRepositoryWithoutSearcher{MovieRepository: base}
	for _, movie := range []model.Movie{
		{Title: "The Matrix", Director: "Wachowski", Genre: "Sci-Fi"},
		{Title: "Unrelated", Director: "Director", Genre: "Drama"},
	} {
		if _, err := repository.Create(ctx, movie); err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}
	searcher := NewMovieSearcher(repository)
	movies, err := searcher.Search(ctx, "matrix", "", 0, 10)
	if err != nil {
		t.Fatalf("Search() error = %v", err)
	}
	if len(movies) != 1 || movies[0].Title != "The Matrix" {
		t.Fatalf("Search() = %#v, want The Matrix", movies)
	}
	genres, err := searcher.Genres(ctx)
	if err != nil {
		t.Fatalf("Genres() error = %v", err)
	}
	if want := []string{"Drama", "Sci-Fi"}; !reflect.DeepEqual(genres, want) {
		t.Fatalf("Genres() = %#v, want %#v", genres, want)
	}
}
