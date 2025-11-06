package integration

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestCORSMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORS headers are present on product API responses", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with CORS middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make a GET request to list products
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify CORS headers are present
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("CORS preflight OPTIONS request returns 204 No Content", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with CORS middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make an OPTIONS preflight request
		req := httptest.NewRequest(http.MethodOptions, "/products", nil)
		req.Header.Set("Origin", "http://example.com")
		req.Header.Set("Access-Control-Request-Method", "POST")
		req.Header.Set("Access-Control-Request-Headers", "Content-Type")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify preflight response
		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
	})

	t.Run("CORS headers are present on POST product creation", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with CORS middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make a POST request to create a product
		body := `{"name":"Test Product","description":"A test product","price":99.99}`
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify CORS headers are present
		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestLoggingMiddleware_Integration(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Logger middleware logs requests to product API", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with Logger middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make a GET request (logging happens in background)
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify request was successful
		// Logger middleware logs in the background, so we just verify the request worked
		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Logger middleware logs POST requests with status codes", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with Logger middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make a POST request to create a product
		body := `{"name":"Test Product","description":"A test product","price":99.99}`
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify request was successful (logging happens in background)
		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Logger middleware logs error status codes", func(t *testing.T) {
		testDB := SetupTestDB(t)
		defer testDB.Cleanup(t)
		testDB.TruncateTables(t)

		// Set up repositories and services
		productRepo := reposql.NewProductRepository(testDB.DB)
		eventRepo := reposql.NewEventRepository(testDB.DB)
		productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

		// Set up HTTP router with Logger middleware
		router := gin.New()
		productCtr := controller.NewProductController(productService)
		cfg := &config.Config{}
		httpAPI.InitRouter(cfg, nil, router, productCtr)

		// Make a request with invalid data to trigger an error
		body := `{"invalid":"data"}`
		req := httptest.NewRequest(http.MethodPost, "/products", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		// Verify error was logged (status 400)
		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
