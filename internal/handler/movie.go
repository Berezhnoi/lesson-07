package handler

import (
	"io"
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"

	"homework/internal/apperr"
	"homework/internal/service"
)

type MovieHandler struct {
	service *service.MovieService
}

func NewMovieHandler(movieService *service.MovieService) *MovieHandler {
	return &MovieHandler{service: movieService}
}

func (handler *MovieHandler) RegisterRoutes(routes *gin.RouterGroup) {
	routes.POST("", handler.create)
	routes.GET("", handler.list)
	routes.GET("/", handler.getByID)
	routes.GET("/:id", handler.getByID)
	routes.PUT("/:id", handler.replace)
	routes.DELETE("/:id", handler.delete)
}

// list godoc
// @Summary List movies
// @Description Returns movies ordered by ascending ID. Filtering is applied before pagination; limits above 100 are capped.
// @Tags movies
// @Accept json
// @Produce json
// @Param page query int false "Page number, starting at 1" minimum(1) default(1)
// @Param limit query int false "Page size, capped at 100" minimum(1) default(10)
// @Param genre query string false "Exact, case-sensitive genre filter"
// @Success 200 {array} model.MovieResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /movies [get]
func (handler *MovieHandler) list(c *gin.Context) {
	page, hasPage := c.GetQuery("page")
	limit, hasLimit := c.GetQuery("limit")
	movies, err := handler.service.ListPage(c.Query("genre"), page, limit, hasPage, hasLimit)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, movies)
}

// create godoc
// @Summary Create a movie
// @Description Creates a movie. Required strings are validated manually; omitted optional fields are null.
// @Tags movies
// @Accept json
// @Produce json
// @Param movie body model.MovieRequest true "Movie to create"
// @Success 201 {object} model.MovieResponse
// @Failure 422 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /movies [post]
func (handler *MovieHandler) create(c *gin.Context) {
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		apperr.WriteError(c, service.ValidationError{Message: "Request body could not be read"})
		return
	}
	movie, err := handler.service.Create(body)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusCreated, movie)
}

// getByID godoc
// @Summary Get a movie
// @Description Returns one movie by its uint32 ID.
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int64 true "Movie ID" minimum(0) maximum(4294967295)
// @Success 200 {object} model.MovieResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /movies/{id} [get]
func (handler *MovieHandler) getByID(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}

	movie, ok := handler.service.GetByID(uint32(id))
	if !ok {
		apperr.WriteError(c, service.NotFoundError{Message: "Movie not found"})
		return
	}
	c.JSON(http.StatusOK, movie)
}

// replace godoc
// @Summary Replace a movie
// @Description Fully replaces a movie. Required fields are replaced, and omitted optional fields become null.
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int64 true "Movie ID" minimum(0) maximum(4294967295)
// @Param movie body model.MovieRequest true "Replacement movie"
// @Success 200 {object} model.MovieResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /movies/{id} [put]
func (handler *MovieHandler) replace(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		apperr.WriteError(c, service.ValidationError{Message: "Request body could not be read"})
		return
	}
	movie, err := handler.service.Replace(uint32(id), body)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, movie)
}

// delete godoc
// @Summary Delete a movie
// @Description Deletes one movie and returns an empty response.
// @Tags movies
// @Accept json
// @Produce json
// @Param id path int64 true "Movie ID" minimum(0) maximum(4294967295)
// @Success 204 "No Content"
// @Failure 400 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /movies/{id} [delete]
func (handler *MovieHandler) delete(c *gin.Context) {
	id, err := parseID(c)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	if err := handler.service.Delete(uint32(id)); err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.Header("Content-Length", "0")
	c.Status(http.StatusNoContent)
}

func parseID(c *gin.Context) (uint32, error) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		return 0, service.InvalidIDError{}
	}
	return uint32(id), nil
}
