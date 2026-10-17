package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/golang-jwt/jwt/v5"

	"homework/internal/auth"
	"homework/internal/model"
)

func TestAuthAndRoleFailuresAbortBeforeHandlers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	manager, err := auth.NewTokenManager("middleware-test-secret-with-at-least-32-bytes", time.Hour)
	if err != nil {
		t.Fatalf("NewTokenManager() error = %v", err)
	}
	validToken, err := manager.Issue(model.User{ID: 1, Role: model.RoleViewer})
	if err != nil {
		t.Fatalf("Issue() error = %v", err)
	}
	expiredToken := jwt.NewWithClaims(jwt.SigningMethodHS256, auth.Claims{
		UserID: 1,
		Role:   model.RoleViewer,
		RegisteredClaims: jwt.RegisteredClaims{
			ExpiresAt: jwt.NewNumericDate(time.Now().Add(-time.Minute)),
		},
	})
	expired, err := expiredToken.SignedString([]byte("middleware-test-secret-with-at-least-32-bytes"))
	if err != nil {
		t.Fatalf("SignedString() error = %v", err)
	}

	tests := []struct {
		name       string
		token      string
		addHeader  bool
		wantStatus int
	}{
		{name: "missing token", wantStatus: http.StatusUnauthorized},
		{name: "bad token", token: "not-a-token", addHeader: true, wantStatus: http.StatusUnauthorized},
		{name: "expired token", token: expired, addHeader: true, wantStatus: http.StatusUnauthorized},
		{name: "wrong role", token: validToken, addHeader: true, wantStatus: http.StatusForbidden},
	}

	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			calls := 0
			router := gin.New()
			router.Use(AuthMiddleware(manager), RequireRole(model.RoleAdmin))
			router.GET("/protected", func(c *gin.Context) {
				calls++
				c.Status(http.StatusNoContent)
			})

			request := httptest.NewRequest(http.MethodGet, "/protected", nil)
			if test.addHeader {
				request.Header.Set("Authorization", "Bearer "+test.token)
			}
			response := httptest.NewRecorder()
			router.ServeHTTP(response, request)

			if calls != 0 {
				t.Errorf("sentinel handler called %d times, want 0", calls)
			}
			if response.Code != test.wantStatus {
				t.Errorf("status = %d, want %d (body: %s)", response.Code, test.wantStatus, response.Body)
			}
		})
	}
}
