package sql

import (
	"context"
	"database/sql"
	"fmt"

	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

// TransactionalRepository provides methods to work with multiple repositories in a single transaction
type TransactionalRepository struct {
	db *sql.DB
}

// NewTransactionalRepository creates a new TransactionalRepository
func NewTransactionalRepository(db *sql.DB) *TransactionalRepository {
	return &TransactionalRepository{db: db}
}

// CreateProductWithEvent creates a product and an event in a single transaction
// The eventDataUpdater function is called with the created product to update the event data
func (tr *TransactionalRepository) CreateProductWithEvent(ctx context.Context, product *model.Product, event *model.Event) (*model.Product, error) {
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create product repository with transaction
	productRepo := &ProductRepository{
		db:  tr.db,
		txn: tx,
	}

	// Create event repository with transaction
	eventRepo := &EventRepository{
		db:  tr.db,
		txn: tx,
	}

	// Create product
	createdProductRes, err := productRepo.Create(ctx, product)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create product: %w", err)
	}

	createdProduct, ok := createdProductRes.(*model.Product)
	if !ok {
		tx.Rollback()
		return nil, repository.ErrInvalidType
	}

	// Create event
	_, err = eventRepo.Create(ctx, event)
	if err != nil {
		tx.Rollback()
		return nil, fmt.Errorf("failed to create event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return nil, fmt.Errorf("failed to commit transaction: %w", err)
	}

	return createdProduct, nil
}

// DeleteProductWithEvent deletes a product and creates a deletion event in a single transaction
func (tr *TransactionalRepository) DeleteProductWithEvent(ctx context.Context, product *model.Product, event *model.Event) error {
	tx, err := tr.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create product repository with transaction
	productRepo := &ProductRepository{
		db:  tr.db,
		txn: tx,
	}

	// Create event repository with transaction
	eventRepo := &EventRepository{
		db:  tr.db,
		txn: tx,
	}

	// Delete product
	if err := productRepo.DeleteByID(ctx, product); err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to delete product: %w", err)
	}

	// Create event
	_, err = eventRepo.Create(ctx, event)
	if err != nil {
		tx.Rollback()
		return fmt.Errorf("failed to create event: %w", err)
	}

	// Commit transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
