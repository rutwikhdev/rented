package middleware

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"

	"github.com/jackc/pgx/v5"
	"github.com/labstack/echo/v4"
	"github.com/redis/go-redis/v9"

	"rented/internal/db"
	"rented/internal/handler"
	"rented/internal/utils"
)

func Auth(queries *db.Queries, rdb *redis.Client, logger *utils.Logger) echo.MiddlewareFunc {
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(c echo.Context) error {
			token := handler.ExtractToken(c)
			if token == "" {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "no token provided"})
			}

			session, err := getSessionFromCache(c.Request().Context(), rdb, token, logger)
			if err == nil {
				c.Set("session", session)
				return next(c)
			}

			session, err = getSessionFromDB(c.Request().Context(), queries, rdb, token, logger)
			if err != nil {
				return c.JSON(http.StatusUnauthorized, map[string]string{"error": "unauthorized"})
			}

			c.Set("session", session)
			return next(c)
		}
	}
}

func getSessionFromCache(ctx context.Context, rdb *redis.Client, token string, logger *utils.Logger) (*handler.SessionData, error) {
	data, err := rdb.Get(ctx, "session:"+token).Bytes()
	if err != nil {
		if !errors.Is(err, redis.Nil) {
			logger.Error("auth middleware: redis get failed", err)
		}
		return nil, err
	}

	var session handler.SessionData
	if err := json.Unmarshal(data, &session); err != nil {
		logger.Error("auth middleware: failed to unmarshal cached session", err)
		return nil, err
	}

	return &session, nil
}

func getSessionFromDB(ctx context.Context, queries *db.Queries, rdb *redis.Client, token string, logger *utils.Logger) (*handler.SessionData, error) {
	row, err := queries.GetSessionByToken(ctx, token)
	if err != nil {
		if !errors.Is(err, pgx.ErrNoRows) {
			logger.Error("auth middleware: failed to get session from db", err)
		}
		return nil, err
	}

	session := &handler.SessionData{
		Token:     row.Token,
		UserID:    strconv.FormatInt(row.UID, 10),
		UserName:  row.UName,
		UserEmail: row.UEmail,
		UserType:  row.UType,
	}

	data, err := json.Marshal(session)
	if err == nil {
		ttl := row.ExpiresAt.Time.Sub(row.CreatedAt.Time)
		if err := rdb.Set(ctx, "session:"+token, data, ttl).Err(); err != nil {
			logger.Error("auth middleware: failed to cache session in redis", err)
		}
	} else {
		logger.Error("auth middleware: failed to marshal session for cache", err)
	}

	return session, nil
}
