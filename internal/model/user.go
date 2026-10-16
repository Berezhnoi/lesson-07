package model

import "time"

const (
	RoleViewer = "viewer"
	RoleEditor = "editor"
	RoleAdmin  = "admin"
)

func IsValidRole(role string) bool {
	switch role {
	case RoleViewer, RoleEditor, RoleAdmin:
		return true
	default:
		return false
	}
}

type User struct {
	ID           uint64    `json:"id" gorm:"primaryKey"`
	Email        string    `json:"email" gorm:"not null"`
	Name         string    `json:"name" gorm:"not null"`
	PasswordHash string    `json:"-" gorm:"column:password_hash;not null"`
	Role         string    `json:"role" gorm:"not null;default:viewer"`
	CreatedAt    time.Time `json:"created_at" gorm:"autoCreateTime:false"`
	UpdatedAt    time.Time `json:"updated_at" gorm:"autoUpdateTime:false"`
}

func (User) TableName() string { return "users" }

type RegisterRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
	Name     string `json:"name"`
}

type LoginRequest struct {
	Email    string `json:"email"`
	Password string `json:"password"`
}

type UpdateRoleRequest struct {
	Role string `json:"role"`
}

type UserResponse struct {
	ID        uint64    `json:"id"`
	Email     string    `json:"email"`
	Name      string    `json:"name"`
	Role      string    `json:"role"`
	CreatedAt time.Time `json:"created_at"`
}

type TokenResponse struct {
	AccessToken string `json:"access_token"`
	TokenType   string `json:"token_type"`
	ExpiresIn   int64  `json:"expires_in"`
}

func (user User) ToResponse() UserResponse {
	return UserResponse{
		ID:        user.ID,
		Email:     user.Email,
		Name:      user.Name,
		Role:      user.Role,
		CreatedAt: user.CreatedAt.UTC(),
	}
}
