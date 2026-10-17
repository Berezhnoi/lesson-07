package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"homework/internal/apperr"
)

func RequireRole(allowed ...string) gin.HandlerFunc {
	allowedSet := make(map[string]struct{}, len(allowed))
	for _, role := range allowed {
		allowedSet[role] = struct{}{}
	}
	return func(c *gin.Context) {
		role, ok := RoleFrom(c)
		if !ok || role == "" {
			abortUnauthorized(c, "authentication required")
			return
		}
		if len(allowedSet) == 0 {
			abortForbidden(c)
			return
		}
		if _, allowedRole := allowedSet[role]; !allowedRole {
			abortForbidden(c)
			return
		}
		c.Next()
	}
}

func abortForbidden(c *gin.Context) {
	apperr.WriteError(c, apperr.HTTPError{
		Status:  http.StatusForbidden,
		Code:    "forbidden",
		Message: "forbidden",
	})
}
