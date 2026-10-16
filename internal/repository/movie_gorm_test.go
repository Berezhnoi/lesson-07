package repository

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"testing"

	"gorm.io/gorm"

	"homework/internal/model"
)

func TestGormRepositoryIntegration(t *testing.T) {
	db := openIntegrationDB(t)

	counter := &queryCounter{}
	if err := db.Callback().Query().After("gorm:query").Register("integration:count_queries", func(*gorm.DB) {
		counter.count.Add(1)
	}); err != nil {
		t.Fatalf("register query counter: %v", err)
	}
	repository := NewGormMovieRepository(db).(*gormMovieRepository)
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

	t.Run("create find optional null and reuse", func(t *testing.T) {
		reset(t)
		first, err := repository.Create(ctx, testMovie("First", "Shared Director", "Shared Genre"))
		if err != nil {
			t.Fatalf("create first movie: %v", err)
		}
		if first.ReleaseYear != nil || first.Synopsis != nil {
			t.Fatalf("omitted optional fields = (%v, %v), want nil", first.ReleaseYear, first.Synopsis)
		}
		second, err := repository.Create(ctx, testMovie("Second", "Shared Director", "Shared Genre"))
		if err != nil {
			t.Fatalf("create second movie: %v", err)
		}
		found, err := repository.GetByIDWithError(ctx, first.ID)
		if err != nil {
			t.Fatalf("find first movie: %v", err)
		}
		if found.Title != first.Title || found.Director != "Shared Director" || found.Genre != "Shared Genre" {
			t.Fatalf("found movie = %#v", found)
		}
		for table, want := range map[string]int64{"genres": 1, "directors": 1} {
			var count int64
			if err := db.Table(table).Count(&count).Error; err != nil {
				t.Fatalf("count %s: %v", table, err)
			}
			if count != want {
				t.Errorf("%s row count = %d, want %d (second id=%d)", table, count, want, second.ID)
			}
		}
	})

	t.Run("update clears optional fields and keeps created time", func(t *testing.T) {
		reset(t)
		year := 2020
		synopsis := "Will be cleared"
		created, err := repository.Create(ctx, model.Movie{
			Title: "Before", Director: "Before Director", Country: "Country", Genre: "Before Genre",
			ReleaseYear: &year, Synopsis: &synopsis,
		})
		if err != nil {
			t.Fatalf("create movie: %v", err)
		}
		updated, err := repository.UpdateWithError(ctx, model.Movie{
			ID: created.ID, Title: "After", Director: "After Director", Country: "New Country", Genre: "After Genre",
		})
		if err != nil {
			t.Fatalf("update movie: %v", err)
		}
		if updated.ReleaseYear != nil || updated.Synopsis != nil {
			t.Fatalf("updated optional fields = (%v, %v), want nil", updated.ReleaseYear, updated.Synopsis)
		}
		if !updated.CreatedAt.Equal(created.CreatedAt) {
			t.Errorf("created_at changed from %s to %s", created.CreatedAt, updated.CreatedAt)
		}
		if !updated.UpdatedAt.After(created.UpdatedAt) {
			t.Errorf("updated_at %s did not advance past %s", updated.UpdatedAt, created.UpdatedAt)
		}
	})

	t.Run("delete cascades join and IDs are not reused", func(t *testing.T) {
		reset(t)
		created, err := repository.Create(ctx, testMovie("Delete me", "Delete Director", "Delete Genre"))
		if err != nil {
			t.Fatalf("create movie: %v", err)
		}
		if err := repository.DeleteWithError(ctx, created.ID); err != nil {
			t.Fatalf("delete movie: %v", err)
		}
		var links int64
		if err := db.Table("movie_directors").Count(&links).Error; err != nil {
			t.Fatalf("count movie_directors: %v", err)
		}
		if links != 0 {
			t.Fatalf("movie_directors rows = %d, want 0", links)
		}
		next, err := repository.Create(ctx, testMovie("Next", "Next Director", "Next Genre"))
		if err != nil {
			t.Fatalf("create after delete: %v", err)
		}
		if next.ID <= created.ID {
			t.Fatalf("ID after delete = %d, want greater than %d", next.ID, created.ID)
		}
	})

	t.Run("list ordering exact genre and pagination", func(t *testing.T) {
		reset(t)
		movies := []model.Movie{
			testMovie("First Drama", "Director One", "Drama"),
			testMovie("Comedy", "Director Two", "Comedy"),
			testMovie("Second Drama", "Director Three", "Drama"),
			testMovie("Lowercase", "Director Four", "drama"),
		}
		created := make([]model.Movie, 0, len(movies))
		for _, movie := range movies {
			saved, err := repository.Create(ctx, movie)
			if err != nil {
				t.Fatalf("create list fixture: %v", err)
			}
			created = append(created, saved)
		}
		all, err := repository.ListWithError(ctx, "", 0, 10)
		if err != nil {
			t.Fatalf("list all: %v", err)
		}
		for index := range created {
			if all[index].ID != created[index].ID {
				t.Errorf("list ID at %d = %d, want %d", index, all[index].ID, created[index].ID)
			}
		}
		drama, err := repository.ListWithError(ctx, "Drama", 0, 10)
		if err != nil {
			t.Fatalf("list exact genre: %v", err)
		}
		if len(drama) != 2 || drama[0].Title != "First Drama" || drama[1].Title != "Second Drama" {
			t.Fatalf("exact Drama result = %#v", drama)
		}
		lowercase, err := repository.ListWithError(ctx, "drama", 0, 10)
		if err != nil {
			t.Fatalf("list lowercase genre: %v", err)
		}
		if len(lowercase) != 1 || lowercase[0].Title != "Lowercase" {
			t.Fatalf("case-sensitive genre result = %#v", lowercase)
		}
		page, err := repository.ListWithError(ctx, "Drama", 1, 1)
		if err != nil {
			t.Fatalf("list second page: %v", err)
		}
		if len(page) != 1 || page[0].Title != "Second Drama" {
			t.Fatalf("second page = %#v", page)
		}
		empty, err := repository.ListWithError(ctx, "Drama", 20, 10)
		if err != nil {
			t.Fatalf("list beyond data: %v", err)
		}
		if empty == nil || len(empty) != 0 {
			t.Fatalf("beyond-data list = %#v, want non-nil empty slice", empty)
		}
	})

	t.Run("cancelled context returns error", func(t *testing.T) {
		reset(t)
		cancelled, cancel := context.WithCancel(ctx)
		cancel()
		if _, err := repository.ListWithError(cancelled, "", 0, 10); err == nil {
			t.Fatal("ListWithError with cancelled context succeeded")
		}
	})

	t.Run("concurrent creates use unique IDs", func(t *testing.T) {
		reset(t)
		const workers = 20
		ids := make(chan uint32, workers)
		errs := make(chan error, workers)
		var wait sync.WaitGroup
		for index := 0; index < workers; index++ {
			wait.Add(1)
			go func(index int) {
				defer wait.Done()
				movie, err := repository.Create(ctx, testMovie(fmt.Sprintf("Concurrent %d", index), "Concurrent Director", "Concurrent Genre"))
				if err != nil {
					errs <- err
					return
				}
				ids <- movie.ID
			}(index)
		}
		wait.Wait()
		close(ids)
		close(errs)
		for err := range errs {
			t.Errorf("concurrent create: %v", err)
		}
		seen := make(map[uint32]struct{}, workers)
		for id := range ids {
			if _, exists := seen[id]; exists {
				t.Errorf("duplicate generated ID %d", id)
			}
			seen[id] = struct{}{}
		}
		if len(seen) != workers {
			t.Fatalf("created %d unique IDs, want %d", len(seen), workers)
		}
	})

	t.Run("list query count is independent of page size", func(t *testing.T) {
		reset(t)
		for index := 0; index < 20; index++ {
			if _, err := repository.Create(ctx, testMovie(fmt.Sprintf("Query Count %03d", index), "Query Director", "Query Genre")); err != nil {
				t.Fatalf("seed movie %d: %v", index, err)
			}
		}
		for index := 20; index < 100; index++ {
			if _, err := repository.Create(ctx, testMovie(fmt.Sprintf("Query Count %03d", index), "Query Director", "Query Genre")); err != nil {
				t.Fatalf("seed movie %d: %v", index, err)
			}
		}
		var firstCount int64
		for _, size := range []int{1, 20, 100} {
			counter.count.Store(0)
			movies, err := repository.ListWithError(ctx, "", 0, size)
			if err != nil {
				t.Fatalf("list %d movies: %v", size, err)
			}
			if len(movies) != size {
				t.Fatalf("list size %d returned %d rows", size, len(movies))
			}
			queries := counter.count.Load()
			t.Logf("preloaded List page_size=%d query_count=%d", size, queries)
			if size == 1 {
				firstCount = queries
			} else if queries != firstCount {
				t.Errorf("query count grew from %d to %d for page size %d", firstCount, queries, size)
			}
		}
		if firstCount > 3 {
			t.Errorf("preloaded List ran %d queries, want at most 3", firstCount)
		}
	})
}

type queryCounter struct {
	count atomic.Int64
}

func testMovie(title, director, genre string) model.Movie {
	return model.Movie{
		Title: title, Director: director, Country: "Test Country", Genre: genre,
	}
}
