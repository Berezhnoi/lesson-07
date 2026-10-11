package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"

	"homework/internal/model"
)

// Health godoc
// @Summary Health check
// @Description Reports whether the API process is responding.
// @Tags health
// @Accept json
// @Produce json
// @Success 200 {object} model.HealthResponse
// @Failure 500 {object} model.ErrorResponse
// @Router /health [get]
func Health(c *gin.Context) {
	c.JSON(http.StatusOK, model.HealthResponse{Status: "ok"})
}
