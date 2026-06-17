package handler

import (
	"context"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/jackc/pgx/v5/pgtype"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"rented/internal/db"
	"rented/internal/rules"
	"rented/internal/utils"
)

type queryStore interface {
	CreateProperty(context.Context, db.CreatePropertyParams) (db.Property, error)
	CreateReservation(context.Context, db.CreateReservationParams) (db.Reservation, error)
	CreateSession(context.Context, db.CreateSessionParams) (db.Session, error)
	CreateUser(context.Context, db.CreateUserParams) (db.CreateUserRow, error)
	DeleteSession(context.Context, string) error
	GetPropertyByID(context.Context, pgtype.UUID) (db.Property, error)
	GetUserByEmail(context.Context, string) (db.User, error)
	ListManagerReservationsFiltered(
		context.Context,
		db.ListManagerReservationsFilteredParams,
	) ([]db.ListManagerReservationsFilteredRow, error)
	ListPropertiesByOwner(
		context.Context,
		db.ListPropertiesByOwnerParams,
	) ([]db.ListPropertiesByOwnerRow, error)
}

type sessionCache interface {
	set(context.Context, string, any, time.Duration) error
	del(context.Context, string) error
}

type redisSessionCache struct {
	client *redis.Client
}

func (c redisSessionCache) set(ctx context.Context, key string, value any, ttl time.Duration) error {
	return c.client.Set(ctx, key, value, ttl).Err()
}

func (c redisSessionCache) del(ctx context.Context, key string) error {
	return c.client.Del(ctx, key).Err()
}

type ruleRunner interface {
	Run(context.Context, rules.RuleInput) []error
}

type Handler struct {
	queries queryStore
	redis   sessionCache
	logger  *utils.Logger
	rules   ruleRunner

	reservationLocks sync.Map
}

func New(queries *db.Queries, rdb *redis.Client, logger *utils.Logger, rulesEngine *rules.Engine) *Handler {
	return &Handler{
		queries: queries,
		redis:   redisSessionCache{client: rdb},
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

func requireSession(c echo.Context) (*SessionData, error) {
	session, ok := c.Get("session").(*SessionData)
	if !ok {
		return nil, errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}
	return session, nil
}

func (h *Handler) sessionUserID(c echo.Context, session *SessionData, operation string) (int64, error) {
	userID, err := strconv.ParseInt(session.UserID, 10, 64)
	if err != nil {
		h.logError(operation+": invalid session user id", err)
		return 0, errorResponse(c, http.StatusUnauthorized, "unauthorized")
	}
	return userID, nil
}

func normalizePage(page int) (int, int32) {
	if page < 1 {
		page = 1
	}
	return page, int32((page - 1) * pageSize)
}

func pgTimestamp(t time.Time) pgtype.Timestamptz {
	return pgtype.Timestamptz{Time: t, Valid: true}
}

func textFilter(value string) pgtype.Text {
	return pgtype.Text{String: value, Valid: value != ""}
}

func (h *Handler) logError(message string, err error, args ...any) {
	if h.logger == nil {
		return
	}
	h.logger.Error(message, err, args...)
}

func (h *Handler) internalError(c echo.Context, message string, err error, args ...any) error {
	h.logError(message, err, args...)
	return errorResponse(c, http.StatusInternalServerError, "internal server error")
}

func calcPages(total, pageSize int) int {
	pages := total / pageSize
	if total%pageSize > 0 {
		pages++
	}
	return pages
}
