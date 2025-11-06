package service_test

import (
	"context"
	"database/sql"
	"encoding/json"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	sqlmock "github.com/DATA-DOG/go-sqlmock"
)

// TestCreateProduct_OutboxPattern verifies that product creation and event creation
// happen within the same transaction (outbox pattern).
func TestCreateProduct_OutboxPattern(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Create repositories
	productRepo := reposql.NewProductRepository(db)
	eventRepo := reposql.NewEventRepository(db)

	// Create service with DB (to enable outbox pattern)
	productService := service.NewProductService(db, productRepo, eventRepo, nil)

	// Expect a transaction to begin
	mock.ExpectBegin()

	// Expect product insertion
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), "Test Product", "Test Description", 99.99, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect event insertion (within same transaction)
	mock.ExpectPrepare("INSERT INTO events").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), "product.created", sqlmock.AnyArg(), string(model.EventStatusPending), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Execute the product creation
	product, err := productService.CreateProduct(ctx, "Test Product", "Test Description", 99.99)

	// Verify results
	require.NoError(t, err)
	assert.NotNil(t, product)
	assert.Equal(t, "Test Product", product.Name)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestDeleteProduct_OutboxPattern verifies that product deletion and event creation
// happen within the same transaction (outbox pattern).
func TestDeleteProduct_OutboxPattern(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()
	productID := uuid.New()

	// Create repositories
	productRepo := reposql.NewProductRepository(db)
	eventRepo := reposql.NewEventRepository(db)

	// Create service with DB (to enable outbox pattern)
	productService := service.NewProductService(db, productRepo, eventRepo, nil)

	// Expect a transaction to begin
	mock.ExpectBegin()

	// Expect product lookup
	now := time.Now()
	rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "created_at", "updated_at"}).
		AddRow(productID, "Test Product", "Test Description", 99.99, now, now)
	mock.ExpectPrepare("SELECT \\* FROM products WHERE id").
		ExpectQuery().
		WithArgs(productID).
		WillReturnRows(rows)

	// Expect product deletion
	mock.ExpectPrepare("DELETE FROM products WHERE id").
		ExpectExec().
		WithArgs(productID).
		WillReturnResult(sqlmock.NewResult(0, 1))

	// Expect event insertion (within same transaction)
	mock.ExpectPrepare("INSERT INTO events").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), "product.deleted", sqlmock.AnyArg(), string(model.EventStatusPending), sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Execute the product deletion
	err = productService.DeleteProduct(ctx, productID)

	// Verify results
	require.NoError(t, err)

	// Verify all expectations were met
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestCreateProduct_OutboxPattern_Rollback verifies that when an error occurs
// during event creation, the entire transaction is rolled back.
func TestCreateProduct_OutboxPattern_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	ctx := context.Background()

	// Create repositories
	productRepo := reposql.NewProductRepository(db)
	eventRepo := reposql.NewEventRepository(db)

	// Create service with DB (to enable outbox pattern)
	productService := service.NewProductService(db, productRepo, eventRepo, nil)

	// Expect a transaction to begin
	mock.ExpectBegin()

	// Expect product insertion to succeed
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), "Test Product", "Test Description", 99.99, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect event insertion to fail
	mock.ExpectPrepare("INSERT INTO events").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), "product.created", sqlmock.AnyArg(), string(model.EventStatusPending), sqlmock.AnyArg(), nil).
		WillReturnError(sql.ErrConnDone)

	// Expect transaction rollback (not commit)
	mock.ExpectRollback()

	// Execute the product creation
	product, err := productService.CreateProduct(ctx, "Test Product", "Test Description", 99.99)

	// Verify that creation failed
	require.Error(t, err)
	assert.Nil(t, product)

	// Verify all expectations were met (including rollback)
	assert.NoError(t, mock.ExpectationsWereMet())
}

// TestEventData_SerializationFormat verifies that event data is properly serialized as ProductMessage.
func TestEventData_SerializationFormat(t *testing.T) {
	msg := sqs.ProductMessage{
		Action:    "created",
		ProductID: uuid.New().String(),
		Name:      "Test Product",
		Price:     99.99,
	}

	eventData, err := json.Marshal(msg)
	require.NoError(t, err)

	// Verify that we can deserialize it back
	var deserializedMsg sqs.ProductMessage
	err = json.Unmarshal(eventData, &deserializedMsg)
	require.NoError(t, err)

	assert.Equal(t, msg.Action, deserializedMsg.Action)
	assert.Equal(t, msg.ProductID, deserializedMsg.ProductID)
	assert.Equal(t, msg.Name, deserializedMsg.Name)
	assert.Equal(t, msg.Price, deserializedMsg.Price)
}
