package controller

import (
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/metrics"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

type ProductController struct {
	repo      repository.Repository
	publisher *sqs.Publisher
}

func NewProductController(repo repository.Repository, publisher *sqs.Publisher) *ProductController {
	return &ProductController{
		repo:      repo,
		publisher: publisher,
	}
}

type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
}

type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

func (pc *ProductController) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	product := &model.Product{
		Name:        req.Name,
		Description: req.Description,
		Price:       req.Price,
	}

	created, err := pc.repo.Create(c.Request.Context(), product)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	createdProduct := created.(*model.Product)

	// Increment metrics
	metrics.ProductsCreated.Inc()

	// Send message to SQS
	if pc.publisher != nil {
		msg := sqs.ProductMessage{
			Action:    "created",
			ProductID: createdProduct.ID.String(),
			Name:      createdProduct.Name,
			Price:     createdProduct.Price,
		}
		if err := pc.publisher.PublishProductMessage(c.Request.Context(), msg); err != nil {
			// Log error but don't fail the request
			slog.Error("Failed to send SQS message", slog.Any("err", err), slog.String("action", "created"), slog.String("product_id", createdProduct.ID.String()))
		}
	}

	c.JSON(http.StatusCreated, toProductResponse(createdProduct))
}

func (pc *ProductController) DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	// Find the product first to get its details for the message
	resource, err := pc.repo.FindByID(c.Request.Context(), id)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	product := resource.(*model.Product)

	// Delete the product
	if err := pc.repo.DeleteByID(c.Request.Context(), product); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete product"})
		return
	}

	// Increment metrics
	metrics.ProductsDeleted.Inc()

	// Send message to SQS
	if pc.publisher != nil {
		msg := sqs.ProductMessage{
			Action:    "deleted",
			ProductID: product.ID.String(),
			Name:      product.Name,
			Price:     product.Price,
		}
		if err := pc.publisher.PublishProductMessage(c.Request.Context(), msg); err != nil {
			// Log error but don't fail the request
			slog.Error("Failed to send SQS message", slog.Any("err", err), slog.String("action", "deleted"), slog.String("product_id", product.ID.String()))
		}
	}

	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}

type ListProductsRequest struct {
	Limit int32  `form:"limit"`
	Token string `form:"token"`
}

type ListProductsResponse struct {
	Products      []ProductResponse `json:"products"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

func (pc *ProductController) ListProducts(c *gin.Context) {
	var req ListProductsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	query := repository.NewQuery()
	if err := query.ApplyPagination(req.Limit, req.Token); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	resources, err := pc.repo.List(c.Request.Context(), *query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	var products []ProductResponse
	for _, resource := range resources {
		product := resource.(*model.Product)
		products = append(products, toProductResponse(product))
	}

	response := ListProductsResponse{
		Products: products,
	}

	// Generate next page token if we have results
	if len(resources) > 0 {
		lastProduct := resources[len(resources)-1].(*model.Product)
		paginator := repository.Paginator{
			LastID:        lastProduct.ID,
			LastCreatedAt: lastProduct.CreatedAt,
		}
		response.NextPageToken = paginator.Encode()
	}

	c.JSON(http.StatusOK, response)
}

func toProductResponse(product *model.Product) ProductResponse {
	return ProductResponse{
		ID:          product.ID.String(),
		Name:        product.Name,
		Description: product.Description,
		Price:       product.Price,
		CreatedAt:   product.CreatedAt.Format("2006-01-02T15:04:05Z07:00"),
		UpdatedAt:   product.UpdatedAt.Format("2006-01-02T15:04:05Z07:00"),
	}
}
