package middleware

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

type Middleware struct {
	config *config.Config
	repo   repository.Repository
}

// New initializes the middleware with the given configuration.
// We don't need ctx here because it always has Gin context.
func New(config *config.Config, repo repository.Repository) *Middleware {
	return &Middleware{
		config: config,
		repo:   repo,
	}
}

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
