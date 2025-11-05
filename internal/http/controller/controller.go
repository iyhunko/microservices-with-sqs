package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

// Controller handles general HTTP requests.
type Controller struct {
	repo   repository.Repository
	config *config.Config
}

// New creates a new Controller with the given configuration and repository.
func New(config *config.Config, repo repository.Repository) *Controller {
	return &Controller{
		config: config,
		repo:   repo,
	}
}

// Ping handles the HTTP GET request for health check endpoint.
func (con *Controller) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
