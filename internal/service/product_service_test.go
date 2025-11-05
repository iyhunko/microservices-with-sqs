package service_test

import (
	"context"
	"testing"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockRepository is a mock implementation of repository.Repository
type MockRepository struct {
	mock.Mock
}

func (m *MockRepository) Create(ctx context.Context, resource repository.Resource) (repository.Resource, error) {
	args := m.Called(ctx, resource)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.Resource), args.Error(1)
}

func (m *MockRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.Resource, error) {
	args := m.Called(ctx, id)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(repository.Resource), args.Error(1)
}

func (m *MockRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	args := m.Called(ctx, id)
	return args.Error(0)
}

func (m *MockRepository) List(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	args := m.Called(ctx, query)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).([]repository.Resource), args.Error(1)
}

func (m *MockRepository) WithinTransaction(ctx context.Context, fn func(repo repository.Repository) error) error {
	args := m.Called(ctx, fn)
	return args.Error(0)
}

func TestCreateProduct(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	product := &model.Product{
		ID:          uuid.New(),
		Name:        "Test Product",
		Description: "Test Description",
		Price:       99.99,
	}

	// Mock WithinTransaction to execute the function immediately
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
	assert.Equal(t, "Test Product", created.Name)
	assert.Equal(t, "Test Description", created.Description)
	assert.Equal(t, 99.99, created.Price)

	mockRepo.AssertExpectations(t)
}

func TestDeleteProduct(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	productID := uuid.New()
	product := &model.Product{
		ID:    productID,
		Name:  "Test Product",
		Price: 99.99,
	}

	// Mock WithinTransaction to execute the function immediately
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

func TestListProducts(t *testing.T) {
	ctx := context.Background()
	mockRepo := new(MockRepository)

	resources := []repository.Resource{
		&model.Product{ID: uuid.New(), Name: "Product 1", Price: 10.0},
		&model.Product{ID: uuid.New(), Name: "Product 2", Price: 20.0},
	}

	query := repository.NewQuery()

	mockRepo.On("List", ctx, *query).Return(resources, nil)

	productService := service.NewProductService(mockRepo, nil)

	results, err := productService.ListProducts(ctx, *query)

	require.NoError(t, err)
	assert.Len(t, results, 2)
	assert.Equal(t, "Product 1", results[0].Name)
	assert.Equal(t, "Product 2", results[1].Name)

	mockRepo.AssertExpectations(t)
}
