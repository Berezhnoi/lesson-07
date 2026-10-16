package repository

import (
	"context"
	"errors"
	"testing"

	"homework/internal/model"
)

func TestInMemoryUserRepository(t *testing.T) {
	tests := []struct {
		name string
		run  func(*testing.T, *InMemoryUserRepository)
	}{
		{
			name: "creates viewer and finds case-insensitive email",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				user := model.User{Email: "  Reader@Example.com ", Name: "Reader", PasswordHash: "hash"}
				if err := repository.Create(context.Background(), &user); err != nil {
					t.Fatalf("Create() error = %v", err)
				}
				if user.ID != 1 || user.Role != model.RoleViewer {
					t.Fatalf("created user = %#v, want ID 1 and viewer role", user)
				}
				found, err := repository.FindByEmail(context.Background(), "reader@example.COM")
				if err != nil {
					t.Fatalf("FindByEmail() error = %v", err)
				}
				if found.ID != user.ID || found.Email != "reader@example.com" {
					t.Fatalf("FindByEmail() = %#v, want normalized created user", found)
				}
			},
		},
		{
			name: "rejects duplicate email without case sensitivity",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				first := model.User{Email: "reader@example.com"}
				if err := repository.Create(context.Background(), &first); err != nil {
					t.Fatalf("Create(first) error = %v", err)
				}
				second := model.User{Email: "READER@example.com"}
				if err := repository.Create(context.Background(), &second); !errors.Is(err, ErrEmailTaken) {
					t.Fatalf("Create(duplicate) error = %v, want ErrEmailTaken", err)
				}
			},
		},
		{
			name: "lists in ID order and returns a non-nil empty slice",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				for _, email := range []string{"second@example.com", "first@example.com"} {
					user := model.User{Email: email}
					if err := repository.Create(context.Background(), &user); err != nil {
						t.Fatalf("Create(%q) error = %v", email, err)
					}
				}
				users, err := repository.List(context.Background(), 0, 10)
				if err != nil {
					t.Fatalf("List() error = %v", err)
				}
				if len(users) != 2 || users[0].ID != 1 || users[1].ID != 2 {
					t.Fatalf("List() IDs = %#v, want [1 2]", users)
				}
				empty, err := repository.List(context.Background(), 10, 10)
				if err != nil {
					t.Fatalf("List(empty) error = %v", err)
				}
				if empty == nil || len(empty) != 0 {
					t.Fatalf("List(empty) = %#v, want non-nil empty slice", empty)
				}
			},
		},
		{
			name: "does not reuse IDs and updates role",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				first := model.User{Email: "first@example.com"}
				if err := repository.Create(context.Background(), &first); err != nil {
					t.Fatalf("Create(first) error = %v", err)
				}
				if _, err := repository.UpdateRole(context.Background(), first.ID, model.RoleEditor); err != nil {
					t.Fatalf("UpdateRole() error = %v", err)
				}
				repository.mu.Lock()
				delete(repository.items, first.ID)
				repository.mu.Unlock()
				next := model.User{Email: "next@example.com"}
				if err := repository.Create(context.Background(), &next); err != nil {
					t.Fatalf("Create(next) error = %v", err)
				}
				if next.ID <= first.ID {
					t.Fatalf("next ID = %d, must be greater than deleted ID %d", next.ID, first.ID)
				}
			},
		},
		{
			name: "returns existing not-found error",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				if _, err := repository.FindByID(context.Background(), 42); !errors.Is(err, ErrNotFound) {
					t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
				}
				if _, err := repository.UpdateRole(context.Background(), 42, model.RoleAdmin); !errors.Is(err, ErrNotFound) {
					t.Fatalf("UpdateRole() error = %v, want ErrNotFound", err)
				}
			},
		},
		{
			name: "rejects invalid role",
			run: func(t *testing.T, repository *InMemoryUserRepository) {
				if _, err := repository.UpdateRole(context.Background(), 1, "owner"); !errors.Is(err, ErrInvalidRole) {
					t.Fatalf("UpdateRole() error = %v, want ErrInvalidRole", err)
				}
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			test.run(t, NewInMemoryUserRepository())
		})
	}
}
