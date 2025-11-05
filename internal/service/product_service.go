package service

import (
	"context"
	"log/slog"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/metrics"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

type ProductService struct {
	repo      repository.Repository
	publisher *sqs.Publisher
}

func NewProductService(repo repository.Repository, publisher *sqs.Publisher) *ProductService {
	return &ProductService{
		repo:      repo,
		publisher: publisher,
	}
}

func (ps *ProductService) CreateProduct(ctx context.Context, name, description string, price float64) (*model.Product, error) {
	product := &model.Product{
		Name:        name,
		Description: description,
		Price:       price,
	}

	created, err := ps.repo.Create(ctx, product)
	if err != nil {
		return nil, err
	}

	createdProduct, ok := created.(*model.Product)
	if !ok {
		return nil, repository.ErrInvalidType
	}

	// Increment metrics
	metrics.ProductsCreated.Inc()

	// Send message to SQS
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

func (ps *ProductService) DeleteProduct(ctx context.Context, id uuid.UUID) error {
	// Find the product first to get its details for the message
	resource, err := ps.repo.FindByID(ctx, id)
	if err != nil {
		return err
	}

	product, ok := resource.(*model.Product)
	if !ok {
		return repository.ErrInvalidType
	}

	// Delete the product
	if err := ps.repo.DeleteByID(ctx, product); err != nil {
		return err
	}

	// Increment metrics
	metrics.ProductsDeleted.Inc()

	// Send message to SQS
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

func (ps *ProductService) ListProducts(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	return ps.repo.List(ctx, query)
}
