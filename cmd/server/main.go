package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/labstack/echo/v4"
	"github.com/labstack/echo/v4/middleware"
	"golang.org/x/time/rate"

	"rented/internal/config"
	"rented/internal/db"
	"rented/internal/handler"
	mw "rented/internal/middleware"
	rdb "rented/internal/redis"
	"rented/internal/rules"
	"rented/internal/utils"
)

func main() {
	cfg, err := config.Load("config.toml")
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	ctx := context.Background()
	pool, err := pgxpool.New(ctx, cfg.Database.URL)
	if err != nil {
		log.Fatalf("failed to connect to database: %v", err)
	}
	defer pool.Close()

	if err := db.Migrate(ctx, pool, "schema.sql"); err != nil {
		log.Fatalf("failed to run migrations: %v", err)
	}

	queries := db.New(pool)

	redisClient, err := rdb.New(cfg.Redis.URL)
	if err != nil {
		log.Fatalf("failed to connect to redis: %v", err)
	}
	defer redisClient.Close()

	logger, err := utils.NewLogger("server.log")
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Close()

	e := echo.New()

	e.Use(middleware.RequestLogger())
	e.Use(middleware.Recover())
	e.Use(middleware.CORS())
	e.Use(middleware.RateLimiterWithConfig(middleware.RateLimiterConfig{
		Store: middleware.NewRateLimiterMemoryStore(rate.Limit(100)),
	}))

	rulesEngine := rules.NewEngine(
		rules.NewCheckInNotInPastRule(),
		rules.NewCheckoutAfterCheckinRule(),
		rules.NewNoOverlapRule(queries),
	)

	h := handler.New(queries, redisClient, logger, rulesEngine)

	// health check (no version needed)
	e.GET("/health", h.Health)

	// public routes
	v1 := e.Group("/api/v1")
	v1.POST("/signup", h.Signup)
	v1.POST("/login", h.Login)

	// authenticated routes
	auth := v1.Group("", mw.Auth(queries, redisClient, logger))
	auth.POST("/logout", h.Logout)
	auth.POST("/property", h.ListProperties)
	auth.POST("/property/new", h.CreateProperty)
	auth.POST("/reservation", h.ListReservations)
	auth.POST("/reservation/new", h.CreateReservation)

	go func() {
		addr := fmt.Sprintf(":%d", cfg.Server.Port)
		if err := e.Start(addr); err != nil && err != http.ErrServerClosed {
			log.Fatalf("shutting down the server: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt)
	<-quit

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()
	if err := e.Shutdown(ctx); err != nil {
		log.Fatal(err)
	}
}
