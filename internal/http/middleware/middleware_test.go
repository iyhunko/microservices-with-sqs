package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

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
