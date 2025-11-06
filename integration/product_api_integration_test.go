package integration

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductAPI_CreateProduct_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	// Set up repositories and services
	productRepo := reposql.NewProductRepository(testDB.DB)
	eventRepo := reposql.NewEventRepository(testDB.DB)
	productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

	// Set up HTTP router
	gin.SetMode(gin.TestMode)
	router := gin.New()
	productCtr := controller.NewProductController(productService)
	cfg := &config.Config{}
	httpAPI.InitRouter(cfg, nil, router, nil, productCtr)

	t.Run("create product successfully", func(t *testing.T) {
		testDB.TruncateTables(t)

		reqBody := map[string]interface{}{
			"name":        "Test Laptop",
			"description": "High-performance laptop",
			"price":       1299.99,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		assert.NotEmpty(t, response["id"])
		assert.Equal(t, "Test Laptop", response["name"])
		assert.Equal(t, "High-performance laptop", response["description"])
		assert.Equal(t, 1299.99, response["price"])
		assert.NotEmpty(t, response["created_at"])
		assert.NotEmpty(t, response["updated_at"])

		// Verify product was saved in database
		productID, err := uuid.Parse(response["id"].(string))
		require.NoError(t, err)

		found, err := productRepo.FindByID(req.Context(), productID)
		require.NoError(t, err)
		assert.NotNil(t, found)
	})

	t.Run("create product with invalid data", func(t *testing.T) {
		testDB.TruncateTables(t)

		reqBody := map[string]interface{}{
			"name": "Test Product",
			// Missing required "price" field
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})

	t.Run("create product with negative price", func(t *testing.T) {
		testDB.TruncateTables(t)

		reqBody := map[string]interface{}{
			"name":  "Invalid Product",
			"price": -10.0,
		}
		body, _ := json.Marshal(reqBody)

		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}

func TestProductAPI_ListProducts_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	productRepo := reposql.NewProductRepository(testDB.DB)
	eventRepo := reposql.NewEventRepository(testDB.DB)
	productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	productCtr := controller.NewProductController(productService)
	cfg := &config.Config{}
	httpAPI.InitRouter(cfg, nil, router, nil, productCtr)

	t.Run("list products", func(t *testing.T) {
		testDB.TruncateTables(t)

		// Create test products
		products := []struct {
			name  string
			price float64
		}{
			{"Product 1", 10.99},
			{"Product 2", 20.99},
			{"Product 3", 30.99},
		}

		for _, p := range products {
			reqBody := map[string]interface{}{
				"name":  p.name,
				"price": p.price,
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		// List products
		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		productsArray := response["products"].([]interface{})
		assert.Len(t, productsArray, 3)

		// Verify products are returned in descending order by created_at
		firstProduct := productsArray[0].(map[string]interface{})
		assert.Equal(t, "Product 3", firstProduct["name"])
	})

	t.Run("list products with pagination", func(t *testing.T) {
		testDB.TruncateTables(t)

		// Create multiple products
		for i := 1; i <= 5; i++ {
			reqBody := map[string]interface{}{
				"name":  fmt.Sprintf("Product %d", i),
				"price": float64(i * 10),
			}
			body, _ := json.Marshal(reqBody)
			req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			require.Equal(t, http.StatusCreated, w.Code)
		}

		// Get first page with limit
		req := httptest.NewRequest(http.MethodGet, "/products?limit=2", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		productsArray := response["products"].([]interface{})
		assert.Len(t, productsArray, 2)
		assert.NotEmpty(t, response["next_page_token"])

		// Get next page using token
		nextToken := response["next_page_token"].(string)
		req = httptest.NewRequest(http.MethodGet, fmt.Sprintf("/products?limit=2&token=%s", nextToken), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		err = json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		productsArray = response["products"].([]interface{})
		assert.Len(t, productsArray, 2)
	})

	t.Run("list products when empty", func(t *testing.T) {
		testDB.TruncateTables(t)

		req := httptest.NewRequest(http.MethodGet, "/products", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var response map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &response)
		require.NoError(t, err)

		productsArray := response["products"].([]interface{})
		assert.Empty(t, productsArray)
	})
}

func TestProductAPI_DeleteProduct_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	productRepo := reposql.NewProductRepository(testDB.DB)
	eventRepo := reposql.NewEventRepository(testDB.DB)
	productService := service.NewProductService(testDB.DB, productRepo, eventRepo, nil)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	productCtr := controller.NewProductController(productService)
	cfg := &config.Config{}
	httpAPI.InitRouter(cfg, nil, router, nil, productCtr)

	t.Run("delete product successfully", func(t *testing.T) {
		testDB.TruncateTables(t)

		// Create a product first
		reqBody := map[string]interface{}{
			"name":  "Product to Delete",
			"price": 25.99,
		}
		body, _ := json.Marshal(reqBody)
		req := httptest.NewRequest(http.MethodPost, "/products", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
		require.Equal(t, http.StatusCreated, w.Code)

		var createResponse map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &createResponse)
		require.NoError(t, err)
		productID := createResponse["id"].(string)

		// Delete the product
		req = httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/products/%s", productID), nil)
		w = httptest.NewRecorder()
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)

		var deleteResponse map[string]interface{}
		err = json.Unmarshal(w.Body.Bytes(), &deleteResponse)
		require.NoError(t, err)
		assert.Equal(t, "product deleted successfully", deleteResponse["message"])

		// Verify product was deleted from database
		id, err := uuid.Parse(productID)
		require.NoError(t, err)
		_, err = productRepo.FindByID(req.Context(), id)
		assert.Error(t, err)
	})

	t.Run("delete non-existent product", func(t *testing.T) {
		testDB.TruncateTables(t)

		nonExistentID := uuid.New().String()
		req := httptest.NewRequest(http.MethodDelete, fmt.Sprintf("/products/%s", nonExistentID), nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNotFound, w.Code)
	})

	t.Run("delete product with invalid ID", func(t *testing.T) {
		testDB.TruncateTables(t)

		req := httptest.NewRequest(http.MethodDelete, "/products/invalid-uuid", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusBadRequest, w.Code)
	})
}
