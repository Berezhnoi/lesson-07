package service

import (
	"context"
	"errors"
	"fmt"

	"homework/internal/model"
	"homework/internal/repository"
)

type UserService struct {
	users repository.UserRepository
}

func NewUserService(users repository.UserRepository) *UserService {
	return &UserService{users: users}
}

func (service *UserService) GetByID(ctx context.Context, id uint64) (model.UserResponse, error) {
	user, err := service.users.FindByID(ctx, id)
	if err != nil {
		return model.UserResponse{}, userRepositoryError(err)
	}
	return user.ToResponse(), nil
}

func (service *UserService) ListPage(ctx context.Context, pageValue, limitValue string, hasPage, hasLimit bool) ([]model.UserResponse, error) {
	page, limit, err := parsePaginationValues(pageValue, limitValue, hasPage, hasLimit)
	if err != nil {
		return nil, err
	}
	total, err := service.users.Count(ctx)
	if err != nil {
		return nil, fmt.Errorf("count users: %w", err)
	}
	offset, empty := paginationOffset(page, limit, total)
	if empty {
		return []model.UserResponse{}, nil
	}
	users, err := service.users.List(ctx, offset, limit)
	if err != nil {
		return nil, fmt.Errorf("list users: %w", err)
	}
	responses := make([]model.UserResponse, 0, len(users))
	for _, user := range users {
		responses = append(responses, user.ToResponse())
	}
	return responses, nil
}

func (service *UserService) UpdateRole(ctx context.Context, id uint64, role string) (model.UserResponse, error) {
	if !model.IsValidRole(role) {
		return model.UserResponse{}, ValidationError{Message: "role must be viewer, editor, or admin"}
	}
	user, err := service.users.FindByID(ctx, id)
	if err != nil {
		return model.UserResponse{}, userRepositoryError(err)
	}
	if user.Role == model.RoleAdmin && role != model.RoleAdmin {
		admins, err := service.users.CountByRole(ctx, model.RoleAdmin)
		if err != nil {
			return model.UserResponse{}, fmt.Errorf("count admins: %w", err)
		}
		if admins <= 1 {
			return model.UserResponse{}, ValidationError{Message: "cannot demote the last remaining admin"}
		}
	}
	updated, err := service.users.UpdateRole(ctx, id, role)
	if err != nil {
		return model.UserResponse{}, userRepositoryError(err)
	}
	return updated.ToResponse(), nil
}

func userRepositoryError(err error) error {
	if errors.Is(err, repository.ErrNotFound) {
		return NotFoundError{Message: "User not found"}
	}
	if err != nil {
		return fmt.Errorf("user repository: %w", err)
	}
	return nil
}
