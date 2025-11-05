package sql

import (
	"context"
	"database/sql"
	"testing"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductRepository_WithinTransaction_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &model.Product{
		Name:        "Test Product",
		Description: "Test Description",
		Price:       99.99,
	}

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect insert within transaction
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), product.Name, product.Description, product.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Execute within transaction
	err = repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
		_, err := txRepo.Create(ctx, product)
		return err
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_WithinTransaction_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	product := &model.Product{
		Name:        "Test Product",
		Description: "Test Description",
		Price:       99.99,
	}

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect insert within transaction to fail
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), product.Name, product.Description, product.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnError(sql.ErrConnDone)

	// Expect transaction rollback due to error
	mock.ExpectRollback()

	// Execute within transaction
	err = repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
		_, err := txRepo.Create(ctx, product)
		return err
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "failed to insert product")
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestProductRepository_WithinTransaction_MultipleOperations(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	product1 := &model.Product{
		Name:        "Product 1",
		Description: "Description 1",
		Price:       99.99,
	}

	product2 := &model.Product{
		Name:        "Product 2",
		Description: "Description 2",
		Price:       149.99,
	}

	// Expect transaction begin
	mock.ExpectBegin()

	// Expect first insert
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), product1.Name, product1.Description, product1.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(1, 1))

	// Expect second insert
	mock.ExpectPrepare("INSERT INTO products").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), product2.Name, product2.Description, product2.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
		WillReturnResult(sqlmock.NewResult(2, 1))

	// Expect transaction commit
	mock.ExpectCommit()

	// Execute multiple operations within transaction
	err = repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
		if _, err := txRepo.Create(ctx, product1); err != nil {
			return err
		}
		if _, err := txRepo.Create(ctx, product2); err != nil {
			return err
		}
		return nil
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
