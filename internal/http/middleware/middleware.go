package middleware

import (
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

// Recovery is a middleware that recovers from panics and returns a 500 Internal Server Error
// instead of crashing the server.
func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				slog.Error("Panic recovered",
					slog.Any("error", err),
					slog.String("path", c.Request.URL.Path),
					slog.String("method", c.Request.Method),
				)
				c.JSON(http.StatusInternalServerError, gin.H{
					"error": "Internal Server Error",
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

// CORS is a middleware that adds Cross-Origin Resource Sharing (CORS) headers to HTTP responses.
func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Writer.Header().Set("Access-Control-Allow-Origin", "*")
		c.Writer.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, PATCH, DELETE, OPTIONS")
		c.Writer.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")
		c.Writer.Header().Set("Access-Control-Max-Age", "86400")

		// Handle preflight requests
		if c.Request.Method == http.MethodOptions {
			c.AbortWithStatus(http.StatusNoContent)
			return
		}

		c.Next()
	}
}

// Logger is a middleware that logs HTTP request details including method, URL path,
// remote IP, response status code, and request duration.
func Logger() gin.HandlerFunc {
	return func(c *gin.Context) {
		// Start timer
		start := time.Now()

		// Process request
		c.Next()

		// Calculate duration
		duration := time.Since(start)

		// Log request details
		slog.Info("HTTP request",
			slog.String("method", c.Request.Method),
			slog.String("path", c.Request.URL.Path),
			slog.String("remote_ip", c.ClientIP()),
			slog.Int("status", c.Writer.Status()),
			slog.Duration("duration", duration),
		)
	}
}
