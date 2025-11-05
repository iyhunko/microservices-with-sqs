package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

// ProductRepository implements the Repository interface for Product entities.
type ProductRepository struct {
	db  *sql.DB
	txn *sql.Tx
}

// NewProductRepository creates a new ProductRepository instance.
func NewProductRepository(db *sql.DB) repository.Repository {
	return &ProductRepository{db: db}
}

// getExecutor returns the active executor (transaction if exists, otherwise db)
func (r *ProductRepository) getExecutor() dbExecutor {
	if r.txn != nil {
		return r.txn
	}
	return r.db
}

// WithinTransaction executes a function within a database transaction
func (r *ProductRepository) WithinTransaction(ctx context.Context, fn func(repo repository.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new repository instance with the transaction
	txRepo := &ProductRepository{
		db:  r.db,
		txn: tx,
	}

	// Execute the function with the transactional repository
	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Create inserts a new product into the database.
func (r *ProductRepository) Create(ctx context.Context, resource repository.Resource) (repository.Resource, error) {
	product, ok := resource.(*model.Product)
	if !ok {
		return nil, errors.New("resource must be a *model.Product")
	}

	// Only initialize metadata if not already set
	if product.ID == uuid.Nil {
		product.InitMeta()
	}

	query := `INSERT INTO products (id, name, description, price, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, product.ID, product.Name, product.Description, product.Price, product.CreatedAt, product.UpdatedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert product: %w", err)
	}

	return product, nil
}

// List retrieves products from the database based on the provided query.
func (r *ProductRepository) List(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT * FROM products WHERE 1=1")

	var args []interface{}
	argIndex := 1

	// Apply pagination
	if query.Paginator != nil {
		queryBuilder.WriteString(fmt.Sprintf(" AND (created_at, id) < ($%d, $%d)", argIndex, argIndex+1))
		args = append(args, query.Paginator.LastCreatedAt, query.Paginator.LastID)
		argIndex += 2
	}

	// Order by created_at DESC, id DESC for consistent pagination
	queryBuilder.WriteString(" ORDER BY created_at DESC, id DESC")

	// Apply limit
	limit := query.Limit
	if limit <= 0 {
		limit = repository.DefaultPaginationLimit
	}
	queryBuilder.WriteString(fmt.Sprintf(" LIMIT $%d", argIndex))
	args = append(args, limit)

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, queryBuilder.String())
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return nil, fmt.Errorf("failed to query products: %w", err)
	}
	defer rows.Close()

	var products []repository.Resource
	for rows.Next() {
		var product model.Product
		err := rows.Scan(&product.ID, &product.Name, &product.Description, &product.Price, &product.CreatedAt, &product.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan product: %w", err)
		}
		products = append(products, &product)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return products, nil
}

// FindByID retrieves a single product by ID.
func (r *ProductRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.Resource, error) {
	query := `SELECT * FROM products WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	var result model.Product
	err = stmt.QueryRowContext(ctx, id).Scan(
		&result.ID, &result.Name, &result.Description, &result.Price, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("product not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query product: %w", err)
	}

	return &result, nil
}

// DeleteByID deletes a product by ID.
func (r *ProductRepository) DeleteByID(ctx context.Context, resource repository.Resource) error {
	product, ok := resource.(*model.Product)
	if !ok {
		return errors.New("resource must be a *model.Product")
	}

	query := `DELETE FROM products WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, product.ID)
	if err != nil {
		return fmt.Errorf("failed to delete product: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("product not found")
	}

	return nil
}
