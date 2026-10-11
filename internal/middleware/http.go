package middleware

import (
	"log"
	"net/http"
	"os"
	"runtime/debug"
	"time"

	"github.com/gin-gonic/gin"
)

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		startedAt := time.Now()
		defer func() {
			if recovered := recover(); recovered != nil {
				// Recovery is outside this middleware; log the eventual status and re-panic to it.
				log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, http.StatusInternalServerError, time.Since(startedAt))
				panic(recovered)
			}
			log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, c.Writer.Status(), time.Since(startedAt))
		}()
		c.Next()
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if recovered := recover(); recovered != nil {
				log.Printf("request panic: %v\n%s", recovered, debug.Stack())
				c.AbortWithStatusJSON(http.StatusInternalServerError, gin.H{
					"error": gin.H{
						"code":    "internal_server_error",
						"message": "internal server error",
					},
				})
			}
		}()
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	allowedOrigin := os.Getenv("ALLOWED_ORIGIN")
	if allowedOrigin == "" {
		allowedOrigin = "*"
	}
	return func(c *gin.Context) {
		startedAt := time.Now()
		c.Header("Access-Control-Allow-Origin", allowedOrigin)
		if c.Request.Method == "OPTIONS" {
			c.Header("Access-Control-Allow-Methods", "GET,POST,PUT,DELETE,OPTIONS")
			c.Header("Access-Control-Allow-Headers", "Content-Type,Authorization")
			c.AbortWithStatus(http.StatusNoContent)
			// The preflight short-circuit does not reach the inner request logger.
			log.Printf("%s %s %d %s", c.Request.Method, c.Request.URL.Path, http.StatusNoContent, time.Since(startedAt))
			return
		}
		c.Next()
	}
}
