package repository

import (
	"context"
	"fmt"
	"reflect"
	"testing"

	"gorm.io/gorm"

	"homework/internal/model"
)

func TestGormMovieSearchIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	counter := &queryCounter{}
	if err := db.Callback().Query().After("gorm:query").Register("search_integration:count_queries", func(*gorm.DB) {
		counter.count.Add(1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	searcher := NewGormMovieRepository(db).(MovieSearcher)
	movieRepository := NewGormMovieRepository(db)
	ctx := context.Background()
	reset := func(t *testing.T) {
		t.Helper()
		counter.count.Store(0)
		if err := db.WithContext(ctx).Exec("TRUNCATE TABLE movies, movie_directors, genres, directors RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("truncate test tables: %v", err)
		}
		counter.count.Store(0)
	}
	t.Cleanup(func() {
		if err := db.Exec("TRUNCATE TABLE movies, movie_directors, genres, directors RESTART IDENTITY CASCADE").Error; err != nil {
			t.Errorf("clean test database: %v", err)
		}
	})

	reset(t)
	for _, movie := range []model.Movie{
		testMovie("The Matrix", "Lana Wachowski", "Sci-Fi"),
		testMovie("Matrix Reloaded", "Lilly Wachowski", "Sci-Fi"),
		testMovie("A_B Story", "Percent% Director", "Drama"),
		testMovie("Small Film", "Another Director", "drama"),
	} {
		if _, err := movieRepository.Create(ctx, movie); err != nil {
			t.Fatalf("create fixture: %v", err)
		}
	}

	tests := []struct {
		name   string
		q      string
		genre  string
		offset int
		limit  int
		want   []string
	}{
		{name: "case insensitive title", q: "mAtRiX", limit: 10, want: []string{"The Matrix", "Matrix Reloaded"}},
		{name: "director match", q: "WACHOWSKI", limit: 10, want: []string{"The Matrix", "Matrix Reloaded"}},
		{name: "percent is literal", q: "%", limit: 10, want: []string{"A_B Story"}},
		{name: "underscore is literal", q: "_", limit: 10, want: []string{"A_B Story"}},
		{name: "genre exact", genre: "Drama", limit: 10, want: []string{"A_B Story"}},
		{name: "genre case sensitive", genre: "drama", limit: 10, want: []string{"Small Film"}},
		{name: "page ordering", q: "matrix", offset: 1, limit: 1, want: []string{"Matrix Reloaded"}},
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

	t.Run("search query count does not scale with rows", func(t *testing.T) {
		reset(t)
		const rowCount = 60
		for index := 0; index < rowCount; index++ {
			if _, err := movieRepository.Create(ctx, testMovie(fmt.Sprintf("Catalog %03d", index), "Catalog Director", "Catalog Genre")); err != nil {
				t.Fatalf("seed movie %d: %v", index, err)
			}
		}
		var reference int64
		for _, size := range []int{1, 20, rowCount} {
			counter.count.Store(0)
			movies, err := searcher.Search(ctx, "", "", 0, size)
			if err != nil {
				t.Fatalf("Search %d movies: %v", size, err)
			}
			if len(movies) != size {
				t.Fatalf("Search size %d returned %d rows", size, len(movies))
			}
			queries := counter.count.Load()
			if size == 1 {
				reference = queries
			} else if queries != reference {
				t.Errorf("query count grew from %d to %d for %d rows", reference, queries, size)
			}
			if queries > 3 {
				t.Errorf("Search ran %d queries for %d rows, want at most 3", queries, size)
			}
		}
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := searcher.Search(cancelled, "", "", 0, 10); err == nil {
			t.Fatal("Search with cancelled context succeeded")
		}
		if _, err := searcher.Genres(cancelled); err == nil {
			t.Fatal("Genres with cancelled context succeeded")
		}
	})
}
