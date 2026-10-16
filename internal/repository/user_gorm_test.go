package repository

import (
	"context"
	"errors"
	"testing"
	"time"

	"homework/internal/model"
)

func TestGormUserRepositoryIntegration(t *testing.T) {
	db := openIntegrationDB(t)
	repository := NewGormUserRepository(db)
	ctx := context.Background()
	reset := func(t *testing.T) {
		t.Helper()
		if err := db.WithContext(ctx).Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error; err != nil {
			t.Fatalf("truncate users: %v", err)
		}
	}
	t.Cleanup(func() {
		if err := db.Exec("TRUNCATE TABLE users RESTART IDENTITY CASCADE").Error; err != nil {
			t.Errorf("clean users: %v", err)
		}
	})

	t.Run("create, find, list and update role", func(t *testing.T) {
		reset(t)
		user := model.User{Email: "  reader@Example.com ", Name: "Reader", PasswordHash: "test-hash"}
		if err := repository.Create(ctx, &user); err != nil {
			t.Fatalf("Create() error = %v", err)
		}
		if user.ID != 1 || user.Role != model.RoleViewer {
			t.Fatalf("created user = %#v, want ID 1 and viewer role", user)
		}
		if user.CreatedAt.Location() != time.UTC || user.CreatedAt.Nanosecond()%1000 != 0 {
			t.Errorf("created_at = %s, want UTC microsecond precision", user.CreatedAt)
		}
		found, err := repository.FindByEmail(ctx, "READER@example.COM")
		if err != nil {
			t.Fatalf("FindByEmail() error = %v", err)
		}
		if found.Email != "reader@example.com" || found.ID != user.ID {
			t.Fatalf("FindByEmail() = %#v, want normalized user", found)
		}
		users, err := repository.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if len(users) != 1 || users[0].ID != user.ID {
			t.Fatalf("List() = %#v, want created user", users)
		}
		updated, err := repository.UpdateRole(ctx, user.ID, model.RoleEditor)
		if err != nil {
			t.Fatalf("UpdateRole() error = %v", err)
		}
		if updated.Role != model.RoleEditor || !updated.UpdatedAt.After(user.UpdatedAt) {
			t.Fatalf("updated user = %#v, want editor with newer updated_at", updated)
		}
	})

	t.Run("maps unique email and missing user errors", func(t *testing.T) {
		reset(t)
		first := model.User{Email: "first@example.com", Name: "First", PasswordHash: "hash"}
		if err := repository.Create(ctx, &first); err != nil {
			t.Fatalf("Create(first) error = %v", err)
		}
		duplicate := model.User{Email: "FIRST@example.com", Name: "Duplicate", PasswordHash: "hash"}
		if err := repository.Create(ctx, &duplicate); !errors.Is(err, ErrEmailTaken) {
			t.Fatalf("Create(duplicate) error = %v, want ErrEmailTaken", err)
		}
		if _, err := repository.FindByID(ctx, 99); !errors.Is(err, ErrNotFound) {
			t.Fatalf("FindByID() error = %v, want ErrNotFound", err)
		}
		if _, err := repository.FindByEmail(ctx, "missing@example.com"); !errors.Is(err, ErrNotFound) {
			t.Fatalf("FindByEmail() error = %v, want ErrNotFound", err)
		}
	})

	t.Run("empty list is non-nil", func(t *testing.T) {
		reset(t)
		users, err := repository.List(ctx, 0, 10)
		if err != nil {
			t.Fatalf("List() error = %v", err)
		}
		if users == nil || len(users) != 0 {
			t.Fatalf("List() = %#v, want non-nil empty slice", users)
		}
	})
}
