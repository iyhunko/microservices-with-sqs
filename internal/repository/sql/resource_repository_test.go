package sql

import (
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/stretchr/testify/assert"
)

func TestGetTableName(t *testing.T) {
	t.Run("should return users table name", func(t *testing.T) {
		// given
		user := &model.User{}

		// when
		tableName := getTableName(user)

		// then
		assert.Equal(t, "users", tableName)
	})
}

func TestGetPrimaryKey(t *testing.T) {
	t.Run("should return id as primary key", func(t *testing.T) {
		// given
		id := uuid.New()
		user := &model.User{ID: id}

		// when
		pkField, pkValue := getPrimaryKey(user)

		// then
		assert.Equal(t, "id", pkField)
		assert.Equal(t, id, pkValue)
	})
}

func TestGetFieldsAndValues(t *testing.T) {
	t.Run("should extract all fields and values from User", func(t *testing.T) {
		// given
		id := uuid.New()
		now := time.Now()
		user := &model.User{
			ID:        id,
			Email:     "test@example.com",
			Password:  "password123",
			Name:      "Test User",
			Region:    "US",
			Status:    "active",
			Role:      "user",
			UpdatedAt: now,
			CreatedAt: now,
		}

		// when
		fields, values := getFieldsAndValues(user)

		// then
		assert.Len(t, fields, 9)
		assert.Len(t, values, 9)
		assert.Contains(t, fields, "id")
		assert.Contains(t, fields, "email")
		assert.Contains(t, fields, "password")
		assert.Contains(t, fields, "name")
		assert.Contains(t, fields, "region")
		assert.Contains(t, fields, "status")
		assert.Contains(t, fields, "role")
		assert.Contains(t, fields, "updated_at")
		assert.Contains(t, fields, "created_at")
	})
}

func TestIsZeroValue(t *testing.T) {
	t.Run("should return true for zero string", func(t *testing.T) {
		assert.True(t, isZeroValue(""))
	})

	t.Run("should return false for non-zero string", func(t *testing.T) {
		assert.False(t, isZeroValue("test"))
	})

	t.Run("should return true for zero int", func(t *testing.T) {
		assert.True(t, isZeroValue(0))
	})

	t.Run("should return false for non-zero int", func(t *testing.T) {
		assert.False(t, isZeroValue(42))
	})

	t.Run("should return true for nil UUID", func(t *testing.T) {
		assert.True(t, isZeroValue(uuid.Nil))
	})

	t.Run("should return false for non-nil UUID", func(t *testing.T) {
		assert.False(t, isZeroValue(uuid.New()))
	})

	t.Run("should return true for zero time", func(t *testing.T) {
		assert.True(t, isZeroValue(time.Time{}))
	})

	t.Run("should return false for non-zero time", func(t *testing.T) {
		assert.False(t, isZeroValue(time.Now()))
	})

	t.Run("should return false for bool true", func(t *testing.T) {
		assert.False(t, isZeroValue(true))
	})

	t.Run("should return true for bool false", func(t *testing.T) {
		assert.True(t, isZeroValue(false))
	})

	t.Run("should return true for nil", func(t *testing.T) {
		assert.True(t, isZeroValue(nil))
	})
}
