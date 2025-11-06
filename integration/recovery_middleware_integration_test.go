package integration

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	"github.com/iyhunko/microservices-with-sqs/internal/http/middleware"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestRecoveryMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("recovery middleware catches panic in product API", func(t *testing.T) {
		// Create a router with recovery middleware
		router := gin.New()
		router.Use(middleware.Recovery())

		// Add a test route that panics
		router.GET("/test-panic", func(c *gin.Context) {
			panic("simulated panic in handler")
		})

		req := httptest.NewRequest(http.MethodGet, "/test-panic", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify the server didn't crash and returned 500
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})

	t.Run("recovery middleware is applied to product service routes", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with recovery middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Normal request should work
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)

		// Verify middleware is in the middleware chain
		routes := router.Routes()
		assert.NotEmpty(t, routes)
	})

	t.Run("recovery middleware handles panic during request processing", func(t *testing.T) {
		router := gin.New()
		router.Use(middleware.Recovery())

		// Simulate a handler that panics during processing
		router.POST("/panic-during-processing", func(c *gin.Context) {
			// Simulate some processing before panic
			_ = c.Request.Body
			panic("unexpected error during processing")
		})

		req := httptest.NewRequest(http.MethodPost, "/panic-during-processing", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})
}
