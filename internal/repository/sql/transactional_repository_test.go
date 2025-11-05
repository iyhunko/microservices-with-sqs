package sql_test

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTransactionalRepository_CreateProductWithEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	txRepo := sql.NewTransactionalRepository(db)
	ctx := context.Background()

	t.Run("successful product and event creation", func(t *testing.T) {
		product := &model.Product{
			Name:        "Test Product",
			Description: "Test Description",
			Price:       99.99,
		}

		eventData := map[string]interface{}{
			"action": "created",
			"name":   "Test Product",
			"price":  99.99,
		}
		eventDataJSON, err := json.Marshal(eventData)
		require.NoError(t, err)

		event := &model.Event{
			EventType: "product.created",
			EventData: eventDataJSON,
			Status:    model.EventStatusPending,
		}

		// Expect transaction begin
		mock.ExpectBegin()

		// Expect product insert
		mock.ExpectPrepare("INSERT INTO products").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), product.Name, product.Description, product.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect event insert
		mock.ExpectPrepare("INSERT INTO events").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), event.EventType, event.EventData, event.Status, sqlmock.AnyArg(), nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect transaction commit
		mock.ExpectCommit()

		result, err := txRepo.CreateProductWithEvent(ctx, product, event)

		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "Test Product", result.Name)
		assert.NotEqual(t, uuid.Nil, result.ID)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rollback on product creation failure", func(t *testing.T) {
		product := &model.Product{
			Name:        "Test Product",
			Description: "Test Description",
			Price:       99.99,
		}

		eventData := json.RawMessage(`{"action":"created"}`)
		event := &model.Event{
			EventType: "product.created",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		// Expect transaction begin
		mock.ExpectBegin()

		// Expect product insert to fail
		mock.ExpectPrepare("INSERT INTO products").
			ExpectExec().
			WillReturnError(sqlmock.ErrCancelled)

		// Expect transaction rollback
		mock.ExpectRollback()

		result, err := txRepo.CreateProductWithEvent(ctx, product, event)

		require.Error(t, err)
		assert.Nil(t, result)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestTransactionalRepository_DeleteProductWithEvent(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	txRepo := sql.NewTransactionalRepository(db)
	ctx := context.Background()

	t.Run("successful product deletion and event creation", func(t *testing.T) {
		productID := uuid.New()
		product := &model.Product{
			ID:    productID,
			Name:  "Test Product",
			Price: 99.99,
		}

		eventData := json.RawMessage(`{"action":"deleted"}`)
		event := &model.Event{
			EventType: "product.deleted",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		// Expect transaction begin
		mock.ExpectBegin()

		// Expect product delete
		mock.ExpectPrepare("DELETE FROM products WHERE id").
			ExpectExec().
			WithArgs(productID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect event insert
		mock.ExpectPrepare("INSERT INTO events").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), event.EventType, event.EventData, event.Status, sqlmock.AnyArg(), nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		// Expect transaction commit
		mock.ExpectCommit()

		err := txRepo.DeleteProductWithEvent(ctx, product, event)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("rollback on event creation failure", func(t *testing.T) {
		productID := uuid.New()
		product := &model.Product{
			ID:    productID,
			Name:  "Test Product",
			Price: 99.99,
		}

		eventData := json.RawMessage(`{"action":"deleted"}`)
		event := &model.Event{
			EventType: "product.deleted",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		// Expect transaction begin
		mock.ExpectBegin()

		// Expect product delete
		mock.ExpectPrepare("DELETE FROM products WHERE id").
			ExpectExec().
			WithArgs(productID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		// Expect event insert to fail
		mock.ExpectPrepare("INSERT INTO events").
			ExpectExec().
			WillReturnError(sqlmock.ErrCancelled)

		// Expect transaction rollback
		mock.ExpectRollback()

		err := txRepo.DeleteProductWithEvent(ctx, product, event)

		require.Error(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
