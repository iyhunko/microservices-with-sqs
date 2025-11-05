package service_test

import (
	"context"
	"errors"
	"testing"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestCreateProduct_TransactionRollback(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	expectedErr := errors.New("database error")

	// Mock WithinTransaction to execute the function and return error
	mockRepo.On("WithinTransaction", ctx, mock.AnythingOfType("func(repository.Repository) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(repository.Repository) error)
			// Execute the function with the mock repository
			fn(mockRepo)
		}).Return(expectedErr)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Product")).Return(nil, expectedErr)

	productService := service.NewProductService(mockRepo, nil)

	created, err := productService.CreateProduct(ctx, "Test Product", "Test Description", 99.99)

	require.Error(t, err)
	assert.Nil(t, created)
	assert.Equal(t, expectedErr, err)

	mockRepo.AssertExpectations(t)
}

func TestDeleteProduct_TransactionRollback(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	productID := uuid.New()
	product := &model.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: 99.99,
	}

	expectedErr := errors.New("database error")

	// Mock WithinTransaction to execute the function and return error
	mockRepo.On("WithinTransaction", ctx, mock.AnythingOfType("func(repository.Repository) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(repository.Repository) error)
			// Execute the function with the mock repository
			fn(mockRepo)
		}).Return(expectedErr)

	mockRepo.On("FindByID", ctx, productID).Return(product, nil)
	mockRepo.On("DeleteByID", ctx, productID).Return(expectedErr)

	productService := service.NewProductService(mockRepo, nil)

	err := productService.DeleteProduct(ctx, productID)

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)

	mockRepo.AssertExpectations(t)
}

func TestCreateProduct_TransactionSuccess(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	product := &model.Product{
		ID:          uuid.New(),
		Name:        "Test Product",
		Description: "Test Description",
		Price:       99.99,
	}

	// Mock WithinTransaction to execute the function successfully
	mockRepo.On("WithinTransaction", ctx, mock.AnythingOfType("func(repository.Repository) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(repository.Repository) error)
			// Execute the function with the mock repository
			fn(mockRepo)
		}).Return(nil)

	mockRepo.On("Create", ctx, mock.AnythingOfType("*model.Product")).Return(product, nil)

	productService := service.NewProductService(mockRepo, nil)

	created, err := productService.CreateProduct(ctx, "Test Product", "Test Description", 99.99)

	require.NoError(t, err)
	assert.NotNil(t, created)
	assert.Equal(t, product.ID, created.ID)
	assert.Equal(t, "Test Product", created.Name)

	mockRepo.AssertExpectations(t)
}

func TestDeleteProduct_TransactionSuccess(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	productID := uuid.New()
	product := &model.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: 99.99,
	}

	// Mock WithinTransaction to execute the function successfully
	mockRepo.On("WithinTransaction", ctx, mock.AnythingOfType("func(repository.Repository) error")).
		Run(func(args mock.Arguments) {
			fn := args.Get(1).(func(repository.Repository) error)
			// Execute the function with the mock repository
			fn(mockRepo)
		}).Return(nil)

	mockRepo.On("FindByID", ctx, productID).Return(product, nil)
	mockRepo.On("DeleteByID", ctx, productID).Return(nil)

	productService := service.NewProductService(mockRepo, nil)

	err := productService.DeleteProduct(ctx, productID)

	require.NoError(t, err)

	mockRepo.AssertExpectations(t)
}
