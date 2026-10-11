// Package app wires the whole HTTP API together.
//
// The automatic tests talk to your API ONLY through NewRouter(), so you are free to
// organise the rest of the code (model, repository, handler, middleware) as you like.
package app

import (
	"net/http"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "homework/docs"
	"homework/internal/apperr"
	"homework/internal/handler"
	"homework/internal/middleware"
	"homework/internal/repository"
	"homework/internal/service"
)

// NewRouter creates a router and a fresh in-memory repository for each call.
func NewRouter() http.Handler {
	return newRouter(false)
}

// NewRouterWithSwagger enables the documentation UI for the executable server.
// Contract tests should continue to use NewRouter.
func NewRouterWithSwagger() http.Handler {
	return newRouter(true)
}

func newRouter(enableSwagger bool) http.Handler {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	// CORS must wrap recovery so panic responses also receive CORS headers.
	router.Use(middleware.CORS(), middleware.Recovery(), middleware.RequestLogger())
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	movieService := service.NewMovieService(repository.NewInMemoryMovieRepository())
	movieHandler := handler.NewMovieHandler(movieService)

	router.GET("/health", handler.Health)
	movieHandler.RegisterRoutes(router.Group("/api/v1/movies"))
	if enableSwagger {
		// Swagger 2 applies BasePath to every path, so this alias serves its health operation.
		router.GET("/api/v1/health", handler.Health)
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
	}
	router.NoRoute(func(c *gin.Context) {
		apperr.WriteError(c, apperr.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: "Route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		apperr.WriteError(c, apperr.HTTPError{Status: http.StatusMethodNotAllowed, Code: "not_found", Message: "Method not allowed"})
	})
	return router
}
