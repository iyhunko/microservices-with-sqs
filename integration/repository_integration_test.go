package integration

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProductRepository_CreateWithTransaction_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	productRepo := reposql.NewProductRepository(testDB.DB)

	t.Run("successful transaction commit", func(t *testing.T) {
		testDB.TruncateTables(t)

		var createdProduct *model.Product

		err := productRepo.WithinTransaction(ctx, func(repo repository.Repository) error {
			product := &model.Product{
				Name:        "Test Product",
				Description: "Test Description",
				Price:       99.99,
			}

			result, err := repo.Create(ctx, product)
			if err != nil {
				return err
			}

			createdProduct = result.(*model.Product)
			return nil
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdProduct.ID)

		// Verify product was committed to database
		found, err := productRepo.FindByID(ctx, createdProduct.ID)
		require.NoError(t, err)
		assert.Equal(t, "Test Product", found.(*model.Product).Name)
	})

	t.Run("transaction rollback on error", func(t *testing.T) {
		testDB.TruncateTables(t)

		var productID uuid.UUID

		err := productRepo.WithinTransaction(ctx, func(repo repository.Repository) error {
			product := &model.Product{
				Name:        "Test Product to Rollback",
				Description: "Should not be committed",
				Price:       49.99,
			}

			result, err := repo.Create(ctx, product)
			if err != nil {
				return err
			}

			productID = result.(*model.Product).ID

			// Force rollback by returning an error
			return errors.New("intentional error to trigger rollback")
		})

		require.Error(t, err)
		assert.Contains(t, err.Error(), "intentional error")

		// Verify product was not committed to database
		_, err = productRepo.FindByID(ctx, productID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})

	t.Run("multiple operations in single transaction", func(t *testing.T) {
		testDB.TruncateTables(t)

		var product1ID, product2ID uuid.UUID

		err := productRepo.WithinTransaction(ctx, func(repo repository.Repository) error {
			// Create first product
			product1 := &model.Product{
				Name:        "Product 1",
				Description: "First product",
				Price:       10.99,
			}
			result1, err := repo.Create(ctx, product1)
			if err != nil {
				return err
			}
			product1ID = result1.(*model.Product).ID

			// Create second product
			product2 := &model.Product{
				Name:        "Product 2",
				Description: "Second product",
				Price:       20.99,
			}
			result2, err := repo.Create(ctx, product2)
			if err != nil {
				return err
			}
			product2ID = result2.(*model.Product).ID

			return nil
		})

		require.NoError(t, err)

		// Verify both products were committed
		found1, err := productRepo.FindByID(ctx, product1ID)
		require.NoError(t, err)
		assert.Equal(t, "Product 1", found1.(*model.Product).Name)

		found2, err := productRepo.FindByID(ctx, product2ID)
		require.NoError(t, err)
		assert.Equal(t, "Product 2", found2.(*model.Product).Name)
	})

	t.Run("transaction with delete operation", func(t *testing.T) {
		testDB.TruncateTables(t)

		// First create a product outside transaction
		product := &model.Product{
			Name:        "Product to Delete",
			Description: "Will be deleted in transaction",
			Price:       30.99,
		}
		result, err := productRepo.Create(ctx, product)
		require.NoError(t, err)
		productID := result.(*model.Product).ID

		// Delete within transaction
		err = productRepo.WithinTransaction(ctx, func(repo repository.Repository) error {
			return repo.DeleteByID(ctx, productID)
		})

		require.NoError(t, err)

		// Verify product was deleted
		_, err = productRepo.FindByID(ctx, productID)
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "not found")
	})
}

func TestEventRepository_CreateWithTransaction_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()
	eventRepo := reposql.NewEventRepository(testDB.DB)

	t.Run("create event in transaction", func(t *testing.T) {
		testDB.TruncateTables(t)

		var createdEvent *model.Event

		err := eventRepo.WithinTransaction(ctx, func(repo repository.Repository) error {
			event := &model.Event{
				EventType: "product.created",
				EventData: []byte(`{"product_id":"123","name":"Test"}`),
				Status:    model.EventStatusPending,
			}

			result, err := repo.Create(ctx, event)
			if err != nil {
				return err
			}

			createdEvent = result.(*model.Event)
			return nil
		})

		require.NoError(t, err)
		assert.NotEqual(t, uuid.Nil, createdEvent.ID)
		assert.Equal(t, model.EventStatusPending, createdEvent.Status)

		// Verify event was committed to database
		found, err := eventRepo.FindByID(ctx, createdEvent.ID)
		require.NoError(t, err)
		assert.Equal(t, "product.created", found.(*model.Event).EventType)
	})
}

func TestRepositoryTransactions_ComplexScenarios_Integration(t *testing.T) {
	testDB := SetupTestDB(t)
	defer testDB.Cleanup(t)

	ctx := context.Background()

	t.Run("nested transaction-like operations with product", func(t *testing.T) {
		testDB.TruncateTables(t)

		productRepo := reposql.NewProductRepository(testDB.DB)
		var createdIDs []uuid.UUID

		// Create multiple products in a transaction
		err := productRepo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
			for i := 1; i <= 3; i++ {
				product := &model.Product{
					Name:        "Batch Product " + string(rune('A'+i-1)),
					Description: "Batch created product",
					Price:       float64(i * 10),
				}

				result, err := txRepo.Create(ctx, product)
				if err != nil {
					return err
				}
				createdIDs = append(createdIDs, result.(*model.Product).ID)
			}
			return nil
		})

		require.NoError(t, err)
		assert.Len(t, createdIDs, 3)

		// Verify all products were created
		for _, id := range createdIDs {
			found, err := productRepo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.NotNil(t, found)
		}
	})

	t.Run("transaction rollback with multiple creates", func(t *testing.T) {
		testDB.TruncateTables(t)

		productRepo := reposql.NewProductRepository(testDB.DB)
		var attemptedIDs []uuid.UUID

		// Attempt to create multiple products but fail
		err := productRepo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
			// Create first product
			product1 := &model.Product{
				Name:  "Product 1",
				Price: 10.0,
			}
			result1, err := txRepo.Create(ctx, product1)
			if err != nil {
				return err
			}
			attemptedIDs = append(attemptedIDs, result1.(*model.Product).ID)

			// Create second product
			product2 := &model.Product{
				Name:  "Product 2",
				Price: 20.0,
			}
			result2, err := txRepo.Create(ctx, product2)
			if err != nil {
				return err
			}
			attemptedIDs = append(attemptedIDs, result2.(*model.Product).ID)

			// Simulate an error after creating products
			return errors.New("simulated error - rollback all")
		})

		require.Error(t, err)

		// Verify no products were committed
		for _, id := range attemptedIDs {
			_, err := productRepo.FindByID(ctx, id)
			assert.Error(t, err)
		}
	})

	t.Run("event repository transaction operations", func(t *testing.T) {
		testDB.TruncateTables(t)

		eventRepo := reposql.NewEventRepository(testDB.DB)
		var eventIDs []uuid.UUID

		// Create multiple events in a transaction
		err := eventRepo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
			for i := 1; i <= 3; i++ {
				event := &model.Event{
					EventType: "test.event",
					EventData: []byte(`{"index":` + string(rune('0'+i)) + `}`),
					Status:    model.EventStatusPending,
				}

				result, err := txRepo.Create(ctx, event)
				if err != nil {
					return err
				}
				eventIDs = append(eventIDs, result.(*model.Event).ID)
			}
			return nil
		})

		require.NoError(t, err)
		assert.Len(t, eventIDs, 3)

		// Verify all events were created
		for _, id := range eventIDs {
			found, err := eventRepo.FindByID(ctx, id)
			require.NoError(t, err)
			assert.NotNil(t, found)
			assert.Equal(t, model.EventStatusPending, found.(*model.Event).Status)
		}
	})
}
