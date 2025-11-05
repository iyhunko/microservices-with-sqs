package controller

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
)

// ProductController handles HTTP requests for product operations.
type ProductController struct {
	productService *service.ProductService
}

// NewProductController creates a new ProductController with the given product service.
func NewProductController(productService *service.ProductService) *ProductController {
	return &ProductController{
		productService: productService,
	}
}

// CreateProductRequest represents the request body for creating a product.
type CreateProductRequest struct {
	Name        string  `json:"name" binding:"required"`
	Description string  `json:"description"`
	Price       float64 `json:"price" binding:"required,gt=0"`
}

// ProductResponse represents the response body for a product.
type ProductResponse struct {
	ID          string  `json:"id"`
	Name        string  `json:"name"`
	Description string  `json:"description"`
	Price       float64 `json:"price"`
	CreatedAt   string  `json:"created_at"`
	UpdatedAt   string  `json:"updated_at"`
}

// CreateProduct handles the HTTP POST request for creating a new product.
func (pc *ProductController) CreateProduct(c *gin.Context) {
	var req CreateProductRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	createdProduct, err := pc.productService.CreateProduct(c.Request.Context(), req.Name, req.Description, req.Price)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create product"})
		return
	}

	c.JSON(http.StatusCreated, toProductResponse(createdProduct))
}

// DeleteProduct handles the HTTP DELETE request for deleting a product by ID.
func (pc *ProductController) DeleteProduct(c *gin.Context) {
	idParam := c.Param("id")
	id, err := uuid.Parse(idParam)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid product ID"})
		return
	}

	if err := pc.productService.DeleteProduct(c.Request.Context(), id); err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "product not found"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "product deleted successfully"})
}

// ListProductsRequest represents the query parameters for listing products.
type ListProductsRequest struct {
	Limit int32  `form:"limit"`
	Token string `form:"token"`
}

// ListProductsResponse represents the response body for listing products.
type ListProductsResponse struct {
	Products      []ProductResponse `json:"products"`
	NextPageToken string            `json:"next_page_token,omitempty"`
}

// ListProducts handles the HTTP GET request for listing products with pagination.
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

	products, err := pc.productService.ListProducts(c.Request.Context(), *query)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list products"})
		return
	}

	var productResponses []ProductResponse
	for _, product := range products {
		productResponses = append(productResponses, toProductResponse(product))
	}

	response := ListProductsResponse{
		Products: productResponses,
	}

	// Generate next page token if we have results
	if len(products) > 0 {
		lastProduct := products[len(products)-1]
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
