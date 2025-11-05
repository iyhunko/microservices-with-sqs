package sql

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		product := &model.Product{
			Name:        "Test Product",
			Description: "Test Description",
			Price:       99.99,
		}

		mock.ExpectPrepare("INSERT INTO products").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), product.Name, product.Description, product.Price, sqlmock.AnyArg(), sqlmock.AnyArg()).
			WillReturnResult(sqlmock.NewResult(1, 1))

		result, err := repo.Create(ctx, product)
		require.NoError(t, err)
		assert.NotNil(t, result)

		createdProduct := result.(*model.Product)
		assert.NotEqual(t, uuid.Nil, createdProduct.ID)
		assert.Equal(t, product.Name, createdProduct.Name)
		assert.False(t, createdProduct.CreatedAt.IsZero())
		assert.False(t, createdProduct.UpdatedAt.IsZero())

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProductRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful find", func(t *testing.T) {
		id := uuid.New()

		now := time.Now()
		rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "created_at", "updated_at"}).
			AddRow(id, "Test Product", "Test Description", 99.99, now, now)

		mock.ExpectPrepare("SELECT \\* FROM products WHERE id = \\$1").
			ExpectQuery().
			WithArgs(id).
			WillReturnRows(rows)

		result, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, result)

		foundProduct := result.(*model.Product)
		assert.Equal(t, id, foundProduct.ID)
		assert.Equal(t, "Test Product", foundProduct.Name)
		assert.Equal(t, 99.99, foundProduct.Price)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("product not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("SELECT \\* FROM products WHERE id = \\$1").
			ExpectQuery().
			WithArgs(id).
			WillReturnError(sql.ErrNoRows)

		result, err := repo.FindByID(ctx, id)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "product not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProductRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	t.Run("list without filters", func(t *testing.T) {
		query := repository.NewQuery()
		query.Limit = 10

		now := time.Now()
		id1 := uuid.New()
		id2 := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "created_at", "updated_at"}).
			AddRow(id1, "Product 1", "Description 1", 99.99, now, now).
			AddRow(id2, "Product 2", "Description 2", 149.99, now, now)

		mock.ExpectPrepare("SELECT \\* FROM products WHERE 1=1 ORDER BY created_at DESC, id DESC LIMIT").
			ExpectQuery().
			WithArgs(10).
			WillReturnRows(rows)

		result, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, result, 2)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("list with pagination", func(t *testing.T) {
		query := repository.NewQuery()
		query.Limit = 10
		lastCreatedAt := time.Now().Add(-1 * time.Hour)
		lastID := uuid.New()
		query.Paginator = &repository.Paginator{
			LastID:        lastID,
			LastCreatedAt: lastCreatedAt,
		}

		now := time.Now()
		id := uuid.New()

		rows := sqlmock.NewRows([]string{"id", "name", "description", "price", "created_at", "updated_at"}).
			AddRow(id, "Product 1", "Description 1", 99.99, now, now)

		mock.ExpectPrepare("SELECT \\* FROM products WHERE 1=1 AND \\(created_at, id\\) < \\(\\$1, \\$2\\) ORDER BY created_at DESC, id DESC LIMIT").
			ExpectQuery().
			WithArgs(lastCreatedAt, lastID, 10).
			WillReturnRows(rows)

		result, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, result, 1)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestProductRepository_DeleteByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewProductRepository(db)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		id := uuid.New()
		product := &model.Product{ID: id}

		mock.ExpectPrepare("DELETE FROM products WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteByID(ctx, product)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("product not found", func(t *testing.T) {
		id := uuid.New()
		product := &model.Product{ID: id}

		mock.ExpectPrepare("DELETE FROM products WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteByID(ctx, product)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "product not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
