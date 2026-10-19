package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"

	"homework/internal/apperr"
	"homework/internal/auth"
)

const (
	userIDContextKey = "user_id"
	roleContextKey   = "role"
)

func AuthMiddleware(tokens *auth.TokenManager) gin.HandlerFunc {
	return func(c *gin.Context) {
		header := c.GetHeader("Authorization")
		if header == "" {
			abortUnauthorized(c, "authentication required")
			return
		}
		scheme, token, found := strings.Cut(header, " ")
		if !found || !strings.EqualFold(scheme, "Bearer") || token == "" || token != strings.TrimSpace(token) {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		claims, err := tokens.Parse(token)
		if err != nil {
			abortUnauthorized(c, "invalid or expired token")
			return
		}
		c.Set(userIDContextKey, claims.UserID)
		c.Set(roleContextKey, claims.Role)
		c.Next()
	}
}

func UserIDFrom(c *gin.Context) (uint, bool) {
	value, ok := c.Get(userIDContextKey)
	if !ok {
		return 0, false
	}
	id, ok := value.(uint64)
	return uint(id), ok
}

func SetPrincipal(c *gin.Context, userID uint64, role string) {
	c.Set(userIDContextKey, userID)
	c.Set(roleContextKey, role)
}

func RoleFrom(c *gin.Context) (string, bool) {
	value, ok := c.Get(roleContextKey)
	if !ok {
		return "", false
	}
	role, ok := value.(string)
	return role, ok
}

func SetDisplayName(c *gin.Context, name string) {
	c.Set("user_name", name)
}

func DisplayNameFrom(c *gin.Context) (string, bool) {
	value, ok := c.Get("user_name")
	if !ok {
		return "", false
	}
	name, ok := value.(string)
	return name, ok
}

func abortUnauthorized(c *gin.Context, message string) {
	apperr.WriteError(c, apperr.HTTPError{
		Status:  http.StatusUnauthorized,
		Code:    "unauthorized",
		Message: message,
	})
}
