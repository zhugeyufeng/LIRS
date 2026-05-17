package main

import (
	"context"
	"log/slog"
	"os"
	"strings"
	"time"

	"github.com/gin-contrib/cors"
	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/redis/go-redis/v9"

	"lirs/apps/server/internal/httpapi"
	"lirs/apps/server/internal/store"
)

func main() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	databaseURL := env("DATABASE_URL", "postgres://lirs:lirs@localhost:5432/lirs?sslmode=disable")
	pool, err := pgxpool.New(ctx, databaseURL)
	if err != nil {
		slog.Error("connect postgres", "error", err)
		os.Exit(1)
	}
	defer pool.Close()

	if err := store.Migrate(ctx, pool); err != nil {
		slog.Error("migrate database", "error", err)
		os.Exit(1)
	}
	if err := store.Seed(ctx, pool); err != nil {
		slog.Error("seed database", "error", err)
		os.Exit(1)
	}

	redisClient := redis.NewClient(&redis.Options{
		Addr:     env("REDIS_ADDR", "localhost:6379"),
		Password: env("REDIS_PASSWORD", ""),
	})
	defer func() {
		if err := redisClient.Close(); err != nil {
			slog.Warn("close redis", "error", err)
		}
	}()

	repo := store.NewRepository(pool, redisClient)
	startMaintenanceWorker(repo)
	router := gin.New()
	router.Use(gin.Recovery(), gin.Logger())
	router.Use(cors.New(cors.Config{
		AllowOrigins:  splitEnv("ALLOWED_ORIGINS", "http://localhost:3000"),
		AllowMethods:  []string{"GET", "POST", "PATCH", "DELETE", "OPTIONS"},
		AllowHeaders:  []string{"Origin", "Content-Type", "Authorization"},
		ExposeHeaders: []string{"Content-Length"},
		// Keep credentials enabled only with explicit origins from ALLOWED_ORIGINS.
		AllowCredentials: true,
		MaxAge:           12 * time.Hour,
	}))

	httpapi.RegisterRoutes(router, repo)

	addr := env("HTTP_ADDR", ":8080")
	slog.Info("starting lirs api", "addr", addr)
	if err := router.Run(addr); err != nil {
		slog.Error("run api", "error", err)
		os.Exit(1)
	}
}

func startMaintenanceWorker(repo *store.Repository) {
	run := func() {
		ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
		defer cancel()
		count, err := repo.ExpireStaleReservations(ctx, 24*time.Hour)
		if err != nil {
			slog.Warn("expire stale reservations", "error", err)
		} else if count > 0 {
			slog.Info("expired stale reservations", "count", count)
		}
		sessionCount, err := repo.CleanupExpiredSessions(ctx)
		if err != nil {
			slog.Warn("cleanup expired sessions", "error", err)
		} else if sessionCount > 0 {
			slog.Info("cleaned expired sessions", "count", sessionCount)
		}
	}
	run()
	go func() {
		ticker := time.NewTicker(10 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			run()
		}
	}()
}

func env(key string, fallback string) string {
	value := strings.TrimSpace(os.Getenv(key))
	if value == "" {
		return fallback
	}
	return value
}

func splitEnv(key string, fallback string) []string {
	raw := env(key, fallback)
	parts := strings.Split(raw, ",")
	values := make([]string, 0, len(parts))
	for _, part := range parts {
		value := strings.TrimSpace(part)
		if value != "" {
			values = append(values, value)
		}
	}
	return values
}
