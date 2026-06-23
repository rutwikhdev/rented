package handler

import (
	"errors"
	"net/http"
	"sync"

	"github.com/jackc/pgx/v5/pgconn"
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

	reservationLocks sync.Map
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

const (
	ErrorCodeInvalidRequestBody      = "INVALID_REQUEST_BODY"
	ErrorCodeInternalServerError     = "INTERNAL_SERVER_ERROR"
	ErrorCodeNoTokenProvided         = "NO_TOKEN_PROVIDED"
	ErrorCodeUnauthorized            = "UNAUTHORIZED"
	ErrorCodeInvalidCredentials      = "INVALID_CREDENTIALS"
	ErrorCodeMissingRequiredFields   = "MISSING_REQUIRED_FIELDS"
	ErrorCodeInvalidEmailFormat      = "INVALID_EMAIL_FORMAT"
	ErrorCodeWeakPassword            = "WEAK_PASSWORD"
	ErrorCodeInvalidUserType         = "INVALID_USER_TYPE"
	ErrorCodeEmailAlreadyExists      = "EMAIL_ALREADY_EXISTS"
	ErrorCodeForbidden               = "FORBIDDEN"
	ErrorCodeInvalidPropertyID       = "INVALID_PROPERTY_ID"
	ErrorCodePropertyNotFound        = "PROPERTY_NOT_FOUND"
	ErrorCodeInvalidReservationID    = "INVALID_RESERVATION_ID"
	ErrorCodeReservationNotFound     = "RESERVATION_NOT_FOUND"
	ErrorCodeInvalidCheckInDateTime  = "INVALID_CHECKIN_DATETIME"
	ErrorCodeInvalidCheckOutDateTime = "INVALID_CHECKOUT_DATETIME"
	ErrorCodeInvalidCheckInFrom      = "INVALID_CHECK_IN_FROM"
	ErrorCodeInvalidCheckOutTo       = "INVALID_CHECK_OUT_TO"
	ErrorCodeTimezoneRequired        = "TIMEZONE_REQUIRED"
	ErrorCodeReservationRuleFailed   = "RESERVATION_RULE_FAILED"
	ErrorCodeOverlappingReservation = "OVERLAPPING_RESERVATION"
)

type errorResponseBody struct {
	Error     string `json:"error"`
	ErrorCode string `json:"error_code"`
}

func ErrorResponse(c echo.Context, status int, errorCode string, message string) error {
	return c.JSON(status, errorResponseBody{
		Error:     message,
		ErrorCode: errorCode,
	})
}

func errorResponse(c echo.Context, status int, errorCode string, message string) error {
	return ErrorResponse(c, status, errorCode, message)
}

func invalidRequestBody(c echo.Context) error {
	return errorResponse(c, http.StatusBadRequest, ErrorCodeInvalidRequestBody, "invalid request body")
}

func (h *Handler) internalError(c echo.Context, message string, err error, args ...any) error {
	h.logger.Error(message, err, args...)
	return errorResponse(c, http.StatusInternalServerError, ErrorCodeInternalServerError, "internal server error")
}

func calcPages(total, pageSize int) int {
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}
	return pages
}

func isExclusionViolation(err error) bool {
	var pgErr *pgconn.PgError
	return errors.As(err, &pgErr) && pgErr.Code == "23P01"
}
