package repository

import (
	"errors"
	"testing"

	"homework/internal/model"
)

func TestCreateStopsAtUint32IDLimit(t *testing.T) {
	repository := NewInMemoryMovieRepository()
	repository.nextID = uint64(^uint32(0))

	lastMovie, err := repository.Create(model.Movie{})
	if err != nil {
		t.Fatalf("create maximum ID: %v", err)
	}
	if lastMovie.ID != ^uint32(0) {
		t.Fatalf("last ID = %d, want %d", lastMovie.ID, ^uint32(0))
	}
	if _, err := repository.Create(model.Movie{}); !errors.Is(err, ErrIDExhausted) {
		t.Fatalf("create after ID exhaustion error = %v, want ErrIDExhausted", err)
	}
}
