package repository

import (
	"context"
	"errors"
	"sort"
	"strings"
	"sync"
	"time"

	"homework/internal/model"
)

type UserRepository interface {
	Create(ctx context.Context, user *model.User) error
	FindByID(ctx context.Context, id uint64) (model.User, error)
	FindByEmail(ctx context.Context, email string) (model.User, error)
	List(ctx context.Context, offset, limit int) ([]model.User, error)
	UpdateRole(ctx context.Context, id uint64, role string) (model.User, error)
}

var (
	ErrEmailTaken      = errors.New("email already taken")
	ErrInvalidRole     = errors.New("invalid user role")
	ErrUserIDExhausted = errors.New("user ID space exhausted")
)

type InMemoryUserRepository struct {
	mu     sync.RWMutex
	items  map[uint64]model.User
	nextID uint64
}

var _ UserRepository = (*InMemoryUserRepository)(nil)

func NewInMemoryUserRepository() *InMemoryUserRepository {
	return &InMemoryUserRepository{items: make(map[uint64]model.User), nextID: 1}
}

func (repository *InMemoryUserRepository) Create(ctx context.Context, user *model.User) error {
	if err := ctx.Err(); err != nil {
		return err
	}
	if user == nil {
		return errors.New("user must not be nil")
	}
	candidate := normalizedUser(*user)
	if candidate.Role == "" {
		candidate.Role = model.RoleViewer
	}
	if !model.IsValidRole(candidate.Role) {
		return ErrInvalidRole
	}

	repository.mu.Lock()
	defer repository.mu.Unlock()
	if repository.nextID == 0 {
		return ErrUserIDExhausted
	}
	for _, existing := range repository.items {
		if existing.Email == candidate.Email {
			return ErrEmailTaken
		}
	}

	now := time.Now().UTC().Truncate(time.Microsecond)
	candidate.ID = repository.nextID
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	repository.nextID++
	repository.items[candidate.ID] = candidate
	*user = cloneUser(candidate)
	return nil
}

func (repository *InMemoryUserRepository) FindByID(ctx context.Context, id uint64) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	user, ok := repository.items[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	return cloneUser(user), nil
}

func (repository *InMemoryUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	email = normalizeEmail(email)
	repository.mu.RLock()
	defer repository.mu.RUnlock()
	for _, user := range repository.items {
		if user.Email == email {
			return cloneUser(user), nil
		}
	}
	return model.User{}, ErrNotFound
}

func (repository *InMemoryUserRepository) List(ctx context.Context, offset, limit int) ([]model.User, error) {
	if err := ctx.Err(); err != nil {
		return nil, err
	}
	if offset < 0 || limit < 0 {
		return nil, errors.New("user list offset and limit must not be negative")
	}
	repository.mu.RLock()
	defer repository.mu.RUnlock()

	ids := make([]uint64, 0, len(repository.items))
	for id := range repository.items {
		ids = append(ids, id)
	}
	sort.Slice(ids, func(i, j int) bool { return ids[i] < ids[j] })
	if offset >= len(ids) || limit == 0 {
		return []model.User{}, nil
	}
	end := len(ids)
	if limit < end-offset {
		end = offset + limit
	}
	users := make([]model.User, 0, end-offset)
	for _, id := range ids[offset:end] {
		users = append(users, cloneUser(repository.items[id]))
	}
	return users, nil
}

func (repository *InMemoryUserRepository) UpdateRole(ctx context.Context, id uint64, role string) (model.User, error) {
	if err := ctx.Err(); err != nil {
		return model.User{}, err
	}
	if !model.IsValidRole(role) {
		return model.User{}, ErrInvalidRole
	}
	repository.mu.Lock()
	defer repository.mu.Unlock()
	user, ok := repository.items[id]
	if !ok {
		return model.User{}, ErrNotFound
	}
	user.Role = role
	user.UpdatedAt = nextUserTimestamp(user.UpdatedAt)
	repository.items[id] = user
	return cloneUser(user), nil
}

func normalizedUser(user model.User) model.User {
	user.Email = normalizeEmail(user.Email)
	return user
}

func normalizeEmail(email string) string {
	return strings.ToLower(strings.TrimSpace(email))
}

func cloneUser(user model.User) model.User {
	return user
}

func nextUserTimestamp(previous time.Time) time.Time {
	now := time.Now().UTC().Truncate(time.Microsecond)
	if !now.After(previous) {
		now = previous.Add(time.Microsecond)
	}
	return now
}
