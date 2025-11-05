package service

import (
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/metrics"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

// ProductService provides business logic for managing products.
type ProductService struct {
	db        *sql.DB
	repo      repository.Repository
	eventRepo repository.Repository
	publisher *sqs.Publisher
}

// NewProductService creates a new ProductService with the given DB, repositories, and SQS publisher.
func NewProductService(db *sql.DB, repo repository.Repository, eventRepo repository.Repository, publisher *sqs.Publisher) *ProductService {
	return &ProductService{
		db:        db,
		repo:      repo,
		eventRepo: eventRepo,
		publisher: publisher,
	}
}

// CreateProduct creates a new product with the provided details and stores an event in the same transaction (outbox pattern).
func (ps *ProductService) CreateProduct(ctx context.Context, name, description string, price float64) (*model.Product, error) {
	var createdProduct *model.Product

	product := &model.Product{
		Name:        name,
		Description: description,
		Price:       price,
	}

	// If DB is available (production), use shared transaction across both repos
	if ps.db != nil && ps.eventRepo != nil {
		// Start a transaction
		tx, err := ps.db.BeginTx(ctx, nil)
		if err != nil {
			return nil, fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					slog.Error("failed to rollback transaction", slog.Any("err", rbErr))
				}
			}
		}()

		// Create transactional repositories
		txProductRepo := reposql.NewProductRepositoryWithTx(ps.db, tx)
		txEventRepo := reposql.NewEventRepositoryWithTx(ps.db, tx)

		// Create product in the transaction
		created, err := txProductRepo.Create(ctx, product)
		if err != nil {
			return nil, err
		}

		var ok bool
		createdProduct, ok = created.(*model.Product)
		if !ok {
			return nil, repository.ErrInvalidType
		}

		// Create event in the same transaction (outbox pattern)
		msg := sqs.ProductMessage{
			Action:    "created",
			ProductID: createdProduct.ID.String(),
			Name:      createdProduct.Name,
			Price:     createdProduct.Price,
		}
		eventData, err := json.Marshal(msg)
		if err != nil {
			return nil, err
		}

		event := &model.Event{
			EventType: "product.created",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		_, err = txEventRepo.Create(ctx, event)
		if err != nil {
			return nil, err
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return nil, fmt.Errorf("failed to commit transaction: %w", err)
		}
	} else {
		// Fallback for tests: use single repository transaction
		err := ps.repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
			created, err := txRepo.Create(ctx, product)
			if err != nil {
				return err
			}

			var ok bool
			createdProduct, ok = created.(*model.Product)
			if !ok {
				return repository.ErrInvalidType
			}

			return nil
		})

		if err != nil {
			return nil, err
		}
	}

	// Increment metrics
	metrics.ProductsCreated.Inc()

	return createdProduct, nil
}

// DeleteProduct deletes a product by ID and stores an event in the same transaction (outbox pattern).
func (ps *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	var product *model.Product

	// If DB is available (production), use shared transaction across both repos
	if ps.db != nil && ps.eventRepo != nil {
		// Start a transaction
		tx, err := ps.db.BeginTx(ctx, nil)
		if err != nil {
			return fmt.Errorf("failed to begin transaction: %w", err)
		}
		defer func() {
			if err != nil {
				if rbErr := tx.Rollback(); rbErr != nil {
					slog.Error("failed to rollback transaction", slog.Any("err", rbErr))
				}
			}
		}()

		// Create transactional repositories
		txProductRepo := reposql.NewProductRepositoryWithTx(ps.db, tx)
		txEventRepo := reposql.NewEventRepositoryWithTx(ps.db, tx)

		// Find the product first to get its details for the message
		resource, err := txProductRepo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		var ok bool
		product, ok = resource.(*model.Product)
		if !ok {
			return repository.ErrInvalidType
		}

		// Delete the product
		if err = txProductRepo.DeleteByID(ctx, id); err != nil {
			return err
		}

		// Create event in the same transaction (outbox pattern)
		msg := sqs.ProductMessage{
			Action:    "deleted",
			ProductID: product.ID.String(),
			Name:      product.Name,
			Price:     product.Price,
		}
		eventData, err := json.Marshal(msg)
		if err != nil {
			return err
		}

		event := &model.Event{
			EventType: "product.deleted",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		_, err = txEventRepo.Create(ctx, event)
		if err != nil {
			return err
		}

		// Commit the transaction
		if err = tx.Commit(); err != nil {
			return fmt.Errorf("failed to commit transaction: %w", err)
		}
	} else {
		// Fallback for tests: use single repository transaction
		err := ps.repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
			// Find the product first to get its details for the message
			resource, err := txRepo.FindByID(ctx, id)
			if err != nil {
				return err
			}

			var ok bool
			product, ok = resource.(*model.Product)
			if !ok {
				return repository.ErrInvalidType
			}

			// Delete the product
			if err := txRepo.DeleteByID(ctx, id); err != nil {
				return err
			}

			return nil
		})

		if err != nil {
			return err
		}
	}

	// Increment metrics
	metrics.ProductsDeleted.Inc()

	return nil
}

// ListProducts retrieves a list of products matching the given query criteria.
func (ps *ProductService) ListProducts(ctx context.Context, query repository.Query) ([]*model.Product, error) {
	resources, err := ps.repo.List(ctx, query)
	if err != nil {
		return nil, err
	}

	products := make([]*model.Product, 0, len(resources))
	for _, resource := range resources {
		product, ok := resource.(*model.Product)
		if !ok {
			return nil, repository.ErrInvalidType
		}
		products = append(products, product)
	}

	return products, nil
}
