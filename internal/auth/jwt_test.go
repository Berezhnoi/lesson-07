package auth

import (
	"strings"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"

	"homework/internal/model"
)

const testJWTSecret = "unit-test-secret-with-at-least-32-bytes"

func TestTokenManager(t *testing.T) {
	tests := []struct {
		name  string
		token func(*testing.T, *TokenManager) string
		valid bool
	}{
		{
			name: "round trip",
			token: func(t *testing.T, manager *TokenManager) string {
				token, err := manager.Issue(model.User{ID: 7, Role: model.RoleEditor})
				if err != nil {
					t.Fatalf("Issue() error = %v", err)
				}
				return token
			},
			valid: true,
		},
		{
			name: "expired",
			token: func(t *testing.T, _ *TokenManager) string {
				return signTestClaims(t, testJWTSecret, Claims{
					UserID: 7, Role: model.RoleViewer,
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute))},
				})
			},
		},
		{
			name: "wrong secret",
			token: func(t *testing.T, manager *TokenManager) string {
				token, err := manager.Issue(model.User{ID: 7, Role: model.RoleViewer})
				if err != nil {
					t.Fatalf("Issue() error = %v", err)
				}
				return token
			},
		},
		{
			name: "tampered payload",
			token: func(t *testing.T, manager *TokenManager) string {
				token, err := manager.Issue(model.User{ID: 7, Role: model.RoleViewer})
				if err != nil {
					t.Fatalf("Issue() error = %v", err)
				}
				parts := strings.Split(token, ".")
				parts[1] = parts[1][:len(parts[1])-1] + "A"
				return strings.Join(parts, ".")
			},
		},
		{
			name: "alg none",
			token: func(t *testing.T, _ *TokenManager) string {
				token := jwt.NewWithClaims(jwt.SigningMethodNone, Claims{
					UserID: 7, Role: model.RoleViewer,
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
				})
				signed, err := token.SignedString(jwt.UnsafeAllowNoneSignatureType)
				if err != nil {
					t.Fatalf("sign none token: %v", err)
				}
				return signed
			},
		},
		{
			name: "other algorithm",
			token: func(t *testing.T, _ *TokenManager) string {
				return signTestClaimsWithMethod(t, jwt.SigningMethodHS512, testJWTSecret, Claims{
					UserID: 7, Role: model.RoleViewer,
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
				})
			},
		},
		{
			name: "missing exp",
			token: func(t *testing.T, _ *TokenManager) string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{"user_id": 7, "role": model.RoleViewer})
				signed, err := token.SignedString([]byte(testJWTSecret))
				if err != nil {
					t.Fatalf("sign no-exp token: %v", err)
				}
				return signed
			},
		},
		{
			name: "missing user id",
			token: func(t *testing.T, _ *TokenManager) string {
				token := jwt.NewWithClaims(jwt.SigningMethodHS256, jwt.MapClaims{
					"role": model.RoleViewer, "exp": time.Now().Add(time.Hour).Unix(),
				})
				signed, err := token.SignedString([]byte(testJWTSecret))
				if err != nil {
					t.Fatalf("sign missing-user token: %v", err)
				}
				return signed
			},
		},
		{
			name: "invalid role",
			token: func(t *testing.T, _ *TokenManager) string {
				return signTestClaims(t, testJWTSecret, Claims{
					UserID: 7, Role: "owner",
					RegisteredClaims: jwt.RegisteredClaims{ExpiresAt: jwt.NewNumericDate(time.Now().Add(time.Hour))},
				})
			},
		},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			manager, err := NewTokenManager(testJWTSecret, 24*time.Hour)
			if err != nil {
				t.Fatalf("NewTokenManager() error = %v", err)
			}
			tokenString := test.token(t, manager)
			if test.name == "wrong secret" {
				manager, err = NewTokenManager("another-unit-test-secret-at-least-32-bytes", 24*time.Hour)
				if err != nil {
					t.Fatalf("NewTokenManager(wrong secret) error = %v", err)
				}
			}
			claims, err := manager.Parse(tokenString)
			if test.valid {
				if err != nil {
					t.Fatalf("Parse() error = %v", err)
				}
				if claims.UserID != 7 || claims.Role != model.RoleEditor || claims.ExpiresAt == nil || claims.IssuedAt == nil {
					t.Fatalf("claims = %#v, want user_id 7, editor, iat and exp", claims)
				}
			} else if err == nil {
				t.Fatalf("Parse() accepted invalid token with claims %#v", claims)
			}
		})
	}
}

func TestNewTokenManagerRejectsShortSecret(t *testing.T) {
	if _, err := NewTokenManager("", time.Hour); err == nil {
		t.Fatal("NewTokenManager() accepted empty secret")
	}
	if _, err := NewTokenManager("short", time.Hour); err == nil {
		t.Fatal("NewTokenManager() accepted short secret")
	}
}

func signTestClaims(t *testing.T, secret string, claims Claims) string {
	t.Helper()
	return signTestClaimsWithMethod(t, jwt.SigningMethodHS256, secret, claims)
}

func signTestClaimsWithMethod(t *testing.T, method jwt.SigningMethod, secret string, claims Claims) string {
	t.Helper()
	signed, err := jwt.NewWithClaims(method, claims).SignedString([]byte(secret))
	if err != nil {
		t.Fatalf("sign test token: %v", err)
	}
	return signed
}
