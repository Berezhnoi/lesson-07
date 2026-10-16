package repository

import (
	"context"
	"errors"
	"fmt"
	"time"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"homework/internal/model"
)

type gormUserRepository struct {
	db *gorm.DB
}

var _ UserRepository = (*gormUserRepository)(nil)

func NewGormUserRepository(db *gorm.DB) UserRepository {
	return &gormUserRepository{db: db}
}

func (repository *gormUserRepository) Create(ctx context.Context, user *model.User) error {
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
	now := time.Now().UTC().Truncate(time.Microsecond)
	candidate.CreatedAt = now
	candidate.UpdatedAt = now
	if err := repository.db.WithContext(ctx).Create(&candidate).Error; err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return ErrEmailTaken
		}
		return fmt.Errorf("create user: %w", err)
	}
	*user = candidate
	return nil
}

func (repository *gormUserRepository) FindByID(ctx context.Context, id uint64) (model.User, error) {
	var user model.User
	if err := repository.db.WithContext(ctx).First(&user, id).Error; err != nil {
		return model.User{}, mapUserDatabaseError("find user", err)
	}
	return normalizeUserTimestamps(user), nil
}

func (repository *gormUserRepository) FindByEmail(ctx context.Context, email string) (model.User, error) {
	var user model.User
	if err := repository.db.WithContext(ctx).
		Where("lower(email) = ?", normalizeEmail(email)).
		First(&user).Error; err != nil {
		return model.User{}, mapUserDatabaseError("find user by email", err)
	}
	return normalizeUserTimestamps(user), nil
}

func (repository *gormUserRepository) List(ctx context.Context, offset, limit int) ([]model.User, error) {
	if offset < 0 || limit < 0 {
		return nil, errors.New("user list offset and limit must not be negative")
	}
	users := make([]model.User, 0)
	if err := repository.db.WithContext(ctx).
		Order("id ASC").
		Offset(offset).
		Limit(limit).
		Find(&users).Error; err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	for i := range users {
		users[i] = normalizeUserTimestamps(users[i])
	}
	return users, nil
}

func (repository *gormUserRepository) UpdateRole(ctx context.Context, id uint64, role string) (model.User, error) {
	if !model.IsValidRole(role) {
		return model.User{}, ErrInvalidRole
	}
	var updated model.User
	err := repository.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var current model.User
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).First(&current, id).Error; err != nil {
			return err
		}
		now := nextUserTimestamp(current.UpdatedAt)
		result := tx.Model(&model.User{}).Where("id = ?", id).Updates(map[string]any{
			"role":       role,
			"updated_at": now,
		})
		if result.Error != nil {
			return result.Error
		}
		if result.RowsAffected == 0 {
			return ErrNotFound
		}
		if err := tx.First(&updated, id).Error; err != nil {
			return err
		}
		return nil
	})
	if err != nil {
		return model.User{}, mapUserDatabaseError("update user role", err)
	}
	return normalizeUserTimestamps(updated), nil
}

func mapUserDatabaseError(operation string, err error) error {
	if errors.Is(err, gorm.ErrRecordNotFound) || errors.Is(err, ErrNotFound) {
		return ErrNotFound
	}
	if errors.Is(err, gorm.ErrDuplicatedKey) {
		return ErrEmailTaken
	}
	return fmt.Errorf("%s: %w", operation, err)
}

func normalizeUserTimestamps(user model.User) model.User {
	user.CreatedAt = user.CreatedAt.UTC()
	user.UpdatedAt = user.UpdatedAt.UTC()
	return user
}
