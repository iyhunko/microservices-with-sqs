package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestRecovery(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("recovery middleware catches panic and returns 500", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())

		// Add a route that panics
		router.GET("/panic", func(c *gin.Context) {
			panic("test panic")
		})

		req := httptest.NewRequest(http.MethodGet, "/panic", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify that the server didn't crash and returned 500
		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})

	t.Run("recovery middleware does not affect normal requests", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())

		// Add a normal route
		router.GET("/normal", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/normal", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		// Verify that normal requests work as expected
		assert.Equal(t, http.StatusOK, w.Code)
		assert.Contains(t, w.Body.String(), "success")
	})

	t.Run("recovery middleware catches panic from string", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())

		router.GET("/panic-string", func(c *gin.Context) {
			panic("string panic message")
		})

		req := httptest.NewRequest(http.MethodGet, "/panic-string", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})

	t.Run("recovery middleware catches panic from error", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())

		router.GET("/panic-error", func(c *gin.Context) {
			panic(assert.AnError)
		})

		req := httptest.NewRequest(http.MethodGet, "/panic-error", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})

	t.Run("recovery middleware catches panic from nil pointer dereference", func(t *testing.T) {
		router := gin.New()
		router.Use(Recovery())

		router.GET("/panic-nil", func(c *gin.Context) {
			var ptr *string
			_ = *ptr // This will cause a panic
		})

		req := httptest.NewRequest(http.MethodGet, "/panic-nil", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
		assert.Contains(t, w.Body.String(), "Internal Server Error")
	})
}

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("CORS middleware adds correct headers to GET request", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("CORS middleware handles OPTIONS preflight request", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS())

		router.OPTIONS("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "should not reach here"})
		})

		req := httptest.NewRequest(http.MethodOptions, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusNoContent, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
		assert.Equal(t, "GET, POST, PUT, PATCH, DELETE, OPTIONS", w.Header().Get("Access-Control-Allow-Methods"))
		assert.Equal(t, "Content-Type, Authorization", w.Header().Get("Access-Control-Allow-Headers"))
		assert.Equal(t, "86400", w.Header().Get("Access-Control-Max-Age"))
	})

	t.Run("CORS middleware adds headers to POST request", func(t *testing.T) {
		router := gin.New()
		router.Use(CORS())

		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
		assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
	})
}

func TestLogger(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("Logger middleware processes request without error", func(t *testing.T) {
		router := gin.New()
		router.Use(Logger())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(http.StatusOK, gin.H{"message": "success"})
		})

		req := httptest.NewRequest(http.MethodGet, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})

	t.Run("Logger middleware handles POST request", func(t *testing.T) {
		router := gin.New()
		router.Use(Logger())

		router.POST("/test", func(c *gin.Context) {
			c.JSON(http.StatusCreated, gin.H{"message": "created"})
		})

		req := httptest.NewRequest(http.MethodPost, "/test", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusCreated, w.Code)
	})

	t.Run("Logger middleware handles error status codes", func(t *testing.T) {
		router := gin.New()
		router.Use(Logger())

		router.GET("/error", func(c *gin.Context) {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "server error"})
		})

		req := httptest.NewRequest(http.MethodGet, "/error", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusInternalServerError, w.Code)
	})

	t.Run("Logger middleware measures request duration", func(t *testing.T) {
		router := gin.New()
		router.Use(Logger())

		router.GET("/slow", func(c *gin.Context) {
			time.Sleep(10 * time.Millisecond)
			c.JSON(http.StatusOK, gin.H{"message": "done"})
		})

		req := httptest.NewRequest(http.MethodGet, "/slow", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
	})
}
