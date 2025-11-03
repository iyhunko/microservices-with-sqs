package controller

import (
	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

type Controller struct {
	repo   repository.Repository
	config *config.Config
}

func New(config *config.Config, repo repository.Repository) *Controller {
	return &Controller{
		config: config,
		repo:   repo,
	}
}

func (con *Controller) Ping(c *gin.Context) {
	c.JSON(200, gin.H{
		"message": "pong",
	})
}
