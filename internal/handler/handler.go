package handler

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"rented/internal/db"
	"rented/internal/rules"
	"rented/internal/utils"
)

type Handler struct {
	queries *db.Queries
	redis   *redis.Client
	logger  *utils.Logger
	rules   *rules.Engine
}

func New(queries *db.Queries, rdb *redis.Client, logger *utils.Logger, rulesEngine *rules.Engine) *Handler {
	return &Handler{
		queries: queries,
		redis:   rdb,
		logger:  logger,
		rules:   rulesEngine,
	}
}

func (h *Handler) Health(c echo.Context) error {
	return c.JSON(http.StatusOK, map[string]string{"status": "ok"})
}

func errorResponse(c echo.Context, status int, message string) error {
	return c.JSON(status, map[string]string{"error": message})
}

func invalidRequestBody(c echo.Context) error {
	return errorResponse(c, http.StatusBadRequest, "invalid request body")
}

func (h *Handler) internalError(c echo.Context, message string, err error, args ...any) error {
	h.logger.Error(message, err, args...)
	return errorResponse(c, http.StatusInternalServerError, "internal server error")
}

func calcPages(total, pageSize int) int {
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}
	return pages
}
