package auth

import (
	"errors"
	"fmt"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"homework/internal/model"
)

type Claims struct {
	UserID uint64 `json:"user_id"`
	Role   string `json:"role"`
	jwt.RegisteredClaims
}

type TokenManager struct {
	secret []byte
	ttl    time.Duration
	parser *jwt.Parser
}

func NewTokenManager(secret string, ttl time.Duration) (*TokenManager, error) {
	if len(secret) < 32 {
		return nil, errors.New("JWT secret must be at least 32 bytes")
	}
	if ttl <= 0 {
		return nil, errors.New("JWT TTL must be greater than zero")
	}
	return &TokenManager{
		secret: []byte(secret),
		ttl:    ttl,
		parser: jwt.NewParser(jwt.WithValidMethods([]string{"HS256"}), jwt.WithExpirationRequired()),
	}, nil
}

func (manager *TokenManager) TTL() time.Duration {
	return manager.ttl
}

func (manager *TokenManager) Issue(user model.User) (string, error) {
	if user.ID == 0 {
		return "", errors.New("JWT user ID must be positive")
	}
	if !model.IsValidRole(user.Role) {
		return "", errors.New("JWT user role is invalid")
	}
	now := time.Now()
	claims := Claims{
		UserID: user.ID,
		Role:   user.Role,
		RegisteredClaims: jwt.RegisteredClaims{
			IssuedAt:  jwt.NewNumericDate(now),
			ExpiresAt: jwt.NewNumericDate(now.Add(manager.ttl)),
		},
	}
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	signed, err := token.SignedString(manager.secret)
	if err != nil {
		return "", fmt.Errorf("sign JWT: %w", err)
	}
	return signed, nil
}

func (manager *TokenManager) Parse(tokenString string) (Claims, error) {
	claims := new(Claims)
	token, err := manager.parser.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (any, error) {
		if token.Method != jwt.SigningMethodHS256 {
			return nil, errors.New("unexpected JWT signing method")
		}
		return manager.secret, nil
	})
	if err != nil {
		return Claims{}, fmt.Errorf("parse JWT: %w", err)
	}
	if !token.Valid {
		return Claims{}, errors.New("invalid JWT")
	}
	if claims.UserID == 0 {
		return Claims{}, errors.New("JWT user_id is required")
	}
	if !model.IsValidRole(claims.Role) {
		return Claims{}, errors.New("JWT role is invalid")
	}
	return *claims, nil
}
