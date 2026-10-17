package handler

import (
	"errors"
	"io"
	"net/http"

	"github.com/gin-gonic/gin"

	"homework/internal/apperr"
	"homework/internal/middleware"
	"homework/internal/model"
	"homework/internal/service"
)

type UserHandler struct {
	service *service.UserService
}

func NewUserHandler(userService *service.UserService) *UserHandler {
	return &UserHandler{service: userService}
}

// me godoc
// @Summary Get the current user
// @Description Returns the authenticated user's public profile.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Success 200 {object} model.UserResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /me [get]
func (handler *UserHandler) me(c *gin.Context) {
	userID, ok := middleware.UserIDFrom(c)
	if !ok {
		apperr.WriteError(c, apperr.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "authentication required"})
		return
	}
	user, err := handler.service.GetByID(c.Request.Context(), uint64(userID))
	if err != nil {
		var notFound service.NotFoundError
		if errors.As(err, &notFound) {
			apperr.WriteError(c, apperr.HTTPError{Status: http.StatusUnauthorized, Code: "unauthorized", Message: "invalid or expired token"})
			return
		}
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}

// listUsers godoc
// @Summary List users
// @Description Returns users ordered by ascending ID. Limits above 100 are capped.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page number, starting at 1" minimum(1) default(1)
// @Param limit query int false "Page size, capped at 100" minimum(1) default(10)
// @Success 200 {array} model.UserResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users [get]
func (handler *UserHandler) list(c *gin.Context) {
	page, hasPage := c.GetQuery("page")
	limit, hasLimit := c.GetQuery("limit")
	users, err := handler.service.ListPage(c.Request.Context(), page, limit, hasPage, hasLimit)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, users)
}

// updateRole godoc
// @Summary Update a user's role
// @Description Sets a user's role. The last remaining admin cannot be demoted.
// @Tags users
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int64 true "User ID" minimum(0)
// @Param role body model.UpdateRoleRequest true "New role"
// @Success 200 {object} model.UserResponse
// @Failure 400 {object} model.ErrorResponse
// @Failure 401 {object} model.ErrorResponse
// @Failure 403 {object} model.ErrorResponse
// @Failure 404 {object} model.ErrorResponse
// @Failure 422 {object} model.ErrorResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /users/{id}/role [put]
func (handler *UserHandler) updateRole(c *gin.Context) {
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
	request, err := decodeAuthRequest[model.UpdateRoleRequest](body)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	user, err := handler.service.UpdateRole(c.Request.Context(), uint64(id), request.Role)
	if err != nil {
		apperr.WriteError(c, err)
		return
	}
	c.JSON(http.StatusOK, user)
}
