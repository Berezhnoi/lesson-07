// Package app wires the whole HTTP API together.
//
// The automatic tests talk to your API ONLY through NewRouter(), so you are free to
// organise the rest of the code (model, repository, handler, middleware) as you like.
package app

import (
	"fmt"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	swaggerFiles "github.com/swaggo/files"
	ginSwagger "github.com/swaggo/gin-swagger"

	_ "homework/docs"
	"homework/internal/apperr"
	"homework/internal/auth"
	"homework/internal/handler"
	webhandler "homework/internal/handler/web"
	"homework/internal/middleware"
	"homework/internal/repository"
	"homework/internal/service"
	webui "homework/web"
)

type Deps struct {
	Movies        repository.MovieRepository
	Catalog       *service.CatalogService
	Users         repository.UserRepository
	Tokens        *auth.TokenManager
	EnableSwagger bool
	Secure        bool
	WebUIEnabled  *bool
}

// NewRouter creates a router and a fresh in-memory repository for each call.
func NewRouter() http.Handler {
	return newRouter(unprotectedDeps(repository.NewInMemoryMovieRepositoryAdapter(), false))
}

// NewRouterWithSwagger enables the documentation UI for the executable server.
// Contract tests should continue to use NewRouter.
func NewRouterWithSwagger() http.Handler {
	return newRouter(unprotectedDeps(repository.NewInMemoryMovieRepositoryAdapter(), true))
}

func NewRouterWithRepository(movieRepository repository.MovieRepository) http.Handler {
	return newRouter(unprotectedDeps(movieRepository, true))
}

func NewRouterWithDeps(deps Deps) http.Handler {
	deps.Secure = true
	if deps.WebUIEnabled == nil {
		enabled := true
		deps.WebUIEnabled = &enabled
	}
	if deps.Movies == nil {
		deps.Movies = repository.NewInMemoryMovieRepositoryAdapter()
	}
	if deps.Users == nil {
		deps.Users = repository.NewInMemoryUserRepository()
	}
	return newRouter(deps)
}

func unprotectedDeps(movieRepository repository.MovieRepository, enableSwagger bool) Deps {
	tokens, err := auth.NewTokenManager("unprotected-in-memory-jwt-secret-32b", 24*time.Hour)
	if err != nil {
		panic(err)
	}
	return Deps{
		Movies:        movieRepository,
		Users:         repository.NewInMemoryUserRepository(),
		Tokens:        tokens,
		EnableSwagger: enableSwagger,
		Secure:        false,
	}
}

func newRouter(deps Deps) http.Handler {
	if deps.Catalog == nil {
		deps.Catalog = service.NewCatalogService(repository.NewMovieSearcher(deps.Movies))
	}
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()
	// CORS must wrap recovery so panic responses also receive CORS headers.
	router.Use(middleware.CORS(), middleware.Recovery(), middleware.RequestLogger(), middleware.RequestBodyLimit())
	router.HandleMethodNotAllowed = true
	router.RedirectTrailingSlash = false
	router.RedirectFixedPath = false

	movieHandler := handler.NewMovieHandler(service.NewMovieService(deps.Movies))
	authHandler := handler.NewAuthHandler(service.NewAuthService(deps.Users, deps.Tokens))
	userHandler := handler.NewUserHandler(service.NewUserService(deps.Users))
	var catalogHandler *webhandler.CatalogHandler
	var staticFS http.FileSystem
	if deps.Secure && *deps.WebUIEnabled {
		renderer, err := webhandler.NewRenderer()
		if err != nil {
			panic(fmt.Errorf("initialize web templates: %w", err))
		}
		files, err := webui.StaticFS()
		if err != nil {
			panic(fmt.Errorf("initialize web static files: %w", err))
		}
		staticFS = http.FS(files)
		catalogHandler = webhandler.NewCatalogHandler(deps.Catalog, renderer)
	}

	router.GET("/health", handler.Health)
	api := router.Group("/api/v1")
	if deps.Secure {
		public := api.Group("")
		protected := api.Group("", middleware.AuthMiddleware(deps.Tokens))
		staff := protected.Group("", middleware.RequireRole("editor", "admin"))
		admin := protected.Group("", middleware.RequireRole("admin"))
		handler.RegisterAPI(public, protected, staff, admin, movieHandler, authHandler, userHandler)
	} else {
		handler.RegisterAPI(api, api, api, api, movieHandler, authHandler, userHandler)
	}
	if deps.EnableSwagger {
		// Swagger 2 applies BasePath to every path, so this alias serves its health operation.
		router.GET("/api/v1/health", handler.Health)
		router.GET("/swagger/*any", ginSwagger.WrapHandler(swaggerFiles.Handler, ginSwagger.URL("/swagger/doc.json")))
	}
	if catalogHandler != nil {
		webRoutes := router.Group("")
		webRoutes.Use(webSecurityHeaders())
		webRoutes.GET("/static/*filepath", gin.WrapH(http.StripPrefix("/static/", http.FileServer(staticFS))))
		webRoutes.GET("/", catalogHandler.Root)
		webRoutes.GET("/catalog", catalogHandler.Catalog)
	}
	router.NoRoute(func(c *gin.Context) {
		apperr.WriteError(c, apperr.HTTPError{Status: http.StatusNotFound, Code: "not_found", Message: "Route not found"})
	})
	router.NoMethod(func(c *gin.Context) {
		apperr.WriteError(c, apperr.HTTPError{Status: http.StatusMethodNotAllowed, Code: "not_found", Message: "Method not allowed"})
	})
	return router
}

func webSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self'; style-src 'self'; img-src 'self' data:; base-uri 'none'; form-action 'self'; frame-ancestors 'none'")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "same-origin")
		c.Next()
	}
}
