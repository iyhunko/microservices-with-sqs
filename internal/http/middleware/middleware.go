package middleware

import (
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
