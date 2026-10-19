package web

import (
	"errors"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"

	"homework/internal/middleware"
	"homework/internal/service"
)

type CatalogHandler struct {
	catalog  *service.CatalogService
	renderer *Renderer
	now      func() time.Time
}

func NewCatalogHandler(catalog *service.CatalogService, renderer *Renderer) *CatalogHandler {
	return &CatalogHandler{catalog: catalog, renderer: renderer, now: time.Now}
}

func (handler *CatalogHandler) Root(c *gin.Context) {
	c.Redirect(http.StatusFound, "/catalog")
}

func (handler *CatalogHandler) Catalog(c *gin.Context) {
	pageValue, hasPage := c.GetQuery("page")
	limitValue, hasLimit := c.GetQuery("limit")
	page, limit, err := service.ParseCatalogPagination(pageValue, limitValue, hasPage, hasLimit)
	if err != nil {
		handler.renderer.Page(c, http.StatusBadRequest, "error", ErrorPage{
			StatusText: "Bad Request",
			Message:    "Please provide a valid page and limit.",
		})
		return
	}
	movies, hasNext, err := handler.catalog.Search(c.Request.Context(), service.SearchParams{
		Q: c.Query("q"), Genre: c.Query("genre"), Page: page, Limit: limit,
	})
	if err != nil {
		var invalidPagination service.InvalidPaginationError
		if errors.As(err, &invalidPagination) {
			handler.renderer.Page(c, http.StatusBadRequest, "error", ErrorPage{
				StatusText: "Bad Request",
				Message:    "Please provide a valid page and limit.",
			})
			return
		}
		var validation service.ValidationError
		if errors.As(err, &validation) {
			handler.renderer.Page(c, http.StatusBadRequest, "error", ErrorPage{
				StatusText: "Bad Request",
				Message:    "Please shorten your search query.",
			})
			return
		}
		log.Printf("load catalog page: %v", err)
		handler.renderer.Page(c, http.StatusInternalServerError, "error", ErrorPage{
			StatusText: "Internal server error",
			Message:    "Please try again later.",
		})
		return
	}
	genres, err := handler.catalog.Genres(c.Request.Context())
	if err != nil {
		log.Printf("load catalog genres: %v", err)
		handler.renderer.Page(c, http.StatusInternalServerError, "error", ErrorPage{
			StatusText: "Internal server error",
			Message:    "Please try again later.",
		})
		return
	}

	now := handler.now()
	view := CatalogPage{
		BasePage: BasePage{Title: "Catalog - Movies"},
		Q:        c.Query("q"),
		Genre:    c.Query("genre"),
		Page:     page,
		Limit:    limit,
		HasPrev:  page > 1,
		HasNext:  hasNext,
	}
	if userID, ok := middleware.UserIDFrom(c); ok && userID > 0 {
		role, _ := middleware.RoleFrom(c)
		name, _ := middleware.DisplayNameFrom(c)
		view.User = &UserView{
			Name: name, RoleLabel: roleLabel(role),
			CanAddMovie: middleware.RoleAllowed(role, "editor", "admin"),
			CanRegister: role == "admin",
		}
	}
	view.Movies = make([]MovieRowView, 0, len(movies))
	for _, movie := range movies {
		row := MapMovie(movie, now)
		if view.User != nil {
			role, _ := middleware.RoleFrom(c)
			row.CanEdit = middleware.RoleAllowed(role, "editor", "admin")
			row.CanDelete = role == "admin"
		}
		view.Movies = append(view.Movies, row)
	}
	for _, genre := range genres {
		view.Genres = append(view.Genres, GenreOption{Value: genre, Label: genre, Selected: genre == view.Genre})
	}
	if view.HasPrev {
		view.PrevURL = pageURL(view.Q, view.Genre, limit, page-1)
	}
	if view.HasNext {
		view.NextURL = pageURL(view.Q, view.Genre, limit, page+1)
	}
	handler.renderer.Page(c, http.StatusOK, "catalog", view)
}

func roleLabel(role string) string {
	if role == "" {
		return ""
	}
	return strings.ToUpper(role[:1]) + role[1:]
}

func pageURL(query, genre string, limit, page int) string {
	values := url.Values{}
	values.Set("page", strconv.Itoa(page))
	values.Set("limit", strconv.Itoa(limit))
	if query != "" {
		values.Set("q", query)
	}
	if genre != "" {
		values.Set("genre", genre)
	}
	return "/catalog?" + values.Encode()
}
