package service

import (
	"context"
	"encoding/json"
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
	repo      repository.Repository
	txRepo    *reposql.TransactionalRepository
	publisher *sqs.Publisher
}

// NewProductService creates a new ProductService with the given repository and SQS publisher.
func NewProductService(repo repository.Repository, publisher *sqs.Publisher) *ProductService {
	return &ProductService{
		repo:      repo,
		publisher: publisher,
	}
}

// NewProductServiceWithOutbox creates a new ProductService with outbox pattern support
func NewProductServiceWithOutbox(repo repository.Repository, txRepo *reposql.TransactionalRepository, publisher *sqs.Publisher) *ProductService {
	return &ProductService{
		repo:      repo,
		txRepo:    txRepo,
		publisher: publisher,
	}
}

// CreateProduct creates a new product with the provided details and publishes a notification.
func (ps *ProductService) CreateProduct(ctx context.Context, name, description string, price float64) (*model.Product, error) {
	var createdProduct *model.Product

	product := &model.Product{
		Name:        name,
		Description: description,
		Price:       price,
	}

	// Use outbox pattern if transactional repository is available
	if ps.txRepo != nil {
		// Prepare event data
		eventData := sqs.ProductMessage{
			Action:    "created",
			ProductID: "", // Will be set after product creation
			Name:      name,
			Price:     price,
		}

		eventDataJSON, err := json.Marshal(eventData)
		if err != nil {
			return nil, err
		}

		event := &model.Event{
			EventType: "product.created",
			EventData: eventDataJSON,
			Status:    model.EventStatusPending,
		}

		// Create product and event in a single transaction
		createdProduct, err = ps.txRepo.CreateProductWithEvent(ctx, product, event)
		if err != nil {
			return nil, err
		}

		// Increment metrics
		metrics.ProductsCreated.Inc()

		return createdProduct, nil
	}

	// Original behavior: execute product creation within a transaction
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

	// Increment metrics
	metrics.ProductsCreated.Inc()

	// Send message to SQS (outside transaction)
	if ps.publisher != nil {
		msg := sqs.ProductMessage{
			Action:    "created",
			ProductID: createdProduct.ID.String(),
			Name:      createdProduct.Name,
			Price:     createdProduct.Price,
		}
		if err := ps.publisher.PublishProductMessage(ctx, msg); err != nil {
			// Log error but don't fail the request
			slog.Error("Failed to send SQS message", slog.Any("err", err), slog.String("action", "created"), slog.String("product_id", createdProduct.ID.String()))
		}
	}

	return createdProduct, nil
}

// DeleteProduct deletes a product by ID and publishes a deletion notification.
func (ps *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	var product *model.Product

	// Use outbox pattern if transactional repository is available
	if ps.txRepo != nil {
		// Find the product first to get its details
		resource, err := ps.repo.FindByID(ctx, id)
		if err != nil {
			return err
		}

		var ok bool
		product, ok = resource.(*model.Product)
		if !ok {
			return repository.ErrInvalidType
		}

		// Prepare event data
		eventData := sqs.ProductMessage{
			Action:    "deleted",
			ProductID: product.ID.String(),
			Name:      product.Name,
			Price:     product.Price,
		}

		eventDataJSON, err := json.Marshal(eventData)
		if err != nil {
			return err
		}

		event := &model.Event{
			EventType: "product.deleted",
			EventData: eventDataJSON,
			Status:    model.EventStatusPending,
		}

		// Delete product and create event in a single transaction
		if err := ps.txRepo.DeleteProductWithEvent(ctx, product, event); err != nil {
			return err
		}

		// Increment metrics
		metrics.ProductsDeleted.Inc()

		return nil
	}

	// Original behavior: execute product deletion within a transaction
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
		if err := txRepo.DeleteByID(ctx, product); err != nil {
			return err
		}

		return nil
	})

	if err != nil {
		return err
	}

	// Increment metrics
	metrics.ProductsDeleted.Inc()

	// Send message to SQS (outside transaction)
	if ps.publisher != nil {
		msg := sqs.ProductMessage{
			Action:    "deleted",
			ProductID: product.ID.String(),
			Name:      product.Name,
			Price:     product.Price,
		}
		if err := ps.publisher.PublishProductMessage(ctx, msg); err != nil {
			// Log error but don't fail the request
			slog.Error("Failed to send SQS message", slog.Any("err", err), slog.String("action", "deleted"), slog.String("product_id", product.ID.String()))
		}
	}

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
