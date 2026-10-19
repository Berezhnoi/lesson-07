package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"homework/internal/apperr"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		role, ok := RoleFrom(c)
		if !ok || role == "" {
			abortUnauthorized(c, "authentication required")
			return
		}
		if !RoleAllowed(role, allowed...) {
			abortForbidden(c)
			return
		}
		c.Next()
	}
}

func RoleAllowed(role string, allowed ...string) bool {
	for _, allowedRole := range allowed {
		if role == allowedRole {
			return true
		}
	}
	return false
}

func abortForbidden(c *gin.Context) {
	apperr.WriteError(c, apperr.HTTPError{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "forbidden",
	})
}
