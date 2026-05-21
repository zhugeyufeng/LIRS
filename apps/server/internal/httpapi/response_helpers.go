package httpapi

import (
	"context"
	"errors"
	"io"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/jackc/pgx/v5"
	"github.com/jackc/pgx/v5/pgconn"
)

type callerFunc[T any] func(context.Context) (T, error)

func caller[T any](fn func(context.Context) (T, error)) callerFunc[T] {
	return fn
}

func get[T any](fn callerFunc[T]) gin.HandlerFunc {
	return func(c *gin.Context) {
		item, err := fn(c.Request.Context())
		respond(c, item, err)
	}
}

func bindJSON(c *gin.Context, input any) bool {
	if err := c.ShouldBindJSON(input); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return false
	}
	return true
}

func bindOptionalJSON(c *gin.Context, input any) bool {
	if c.Request == nil || c.Request.Body == nil || c.Request.ContentLength == 0 {
		return true
	}
	if err := c.ShouldBindJSON(input); err != nil {
		if errors.Is(err, io.EOF) {
			return true
		}
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid json payload"})
		return false
	}
	return true
}

func intQuery(c *gin.Context, key string, fallback int) int {
	raw := strings.TrimSpace(c.Query(key))
	if raw == "" {
		return fallback
	}
	value, err := strconv.Atoi(raw)
	if err != nil {
		return fallback
	}
	return value
}

func writeCSVBOM(c *gin.Context) {
	_, _ = c.Writer.Write([]byte{0xEF, 0xBB, 0xBF})
}

func respond(c *gin.Context, payload any, err error) {
	if err == nil {
		c.JSON(http.StatusOK, payload)
		return
	}
	if errors.Is(err, pgx.ErrNoRows) {
		c.JSON(http.StatusNotFound, gin.H{"error": "resource not found"})
		return
	}
	if status, message, ok := postgresClientError(err); ok {
		c.JSON(status, gin.H{"error": message})
		return
	}
	if message, ok := clientSafeError(err); ok {
		c.JSON(http.StatusBadRequest, gin.H{"error": message})
		return
	}
	slog.Error("api request failed", "method", c.Request.Method, "path", c.FullPath(), "error", err)
	c.JSON(http.StatusInternalServerError, gin.H{"error": publicErrorMessage(err)})
}

func postgresClientError(err error) (int, string, bool) {
	var pgErr *pgconn.PgError
	if !errors.As(err, &pgErr) {
		return 0, "", false
	}
	switch pgErr.Code {
	case "23505":
		return http.StatusConflict, "resource already exists", true
	case "23503":
		return http.StatusBadRequest, "referenced resource not found", true
	default:
		return 0, "", false
	}
}

func clientSafeError(err error) (string, bool) {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		return "", false
	}
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return "", false
	}
	var clientErr interface {
		ClientMessage() string
	}
	if !errors.As(err, &clientErr) {
		return "", false
	}
	message := strings.TrimSpace(clientErr.ClientMessage())
	if message == "" {
		return "", false
	}
	return message, true
}

func publicErrorMessage(err error) string {
	var pgErr *pgconn.PgError
	if errors.As(err, &pgErr) {
		if pgErr.Code != "" && pgErr.Message != "" {
			return strings.TrimSpace(pgErr.Message + " (SQLSTATE " + pgErr.Code + ")")
		}
	}
	message := strings.TrimSpace(err.Error())
	if message == "" {
		return "未知错误"
	}
	return message
}
