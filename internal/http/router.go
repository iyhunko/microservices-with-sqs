package http

import (
	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	"github.com/iyhunko/microservices-with-sqs/internal/http/middleware"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

func InitRouter(_ *config.Config, _ repository.Repository, server *gin.Engine, productCtr *controller.ProductController) *gin.Engine {
	// httpMiddleware := middleware.New(config, repo)

	// Apply recovery middleware globally to prevent panics from crashing the server
	server.Use(middleware.Recovery())

	// Product endpoints
	products := server.Group("/products")
	{
		products.POST("", productCtr.CreateProduct)
		products.GET("", productCtr.ListProducts)
		products.DELETE("/:id", productCtr.DeleteProduct)
	}

	return server
}
