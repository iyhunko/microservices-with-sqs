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
	"github.com/lib/pq"
)

// UserRepository implements the Repository interface for User entities.
type UserRepository struct {
	db  *sql.DB
	txn *sql.Tx
}

// NewUserRepository creates a new UserRepository instance.
func NewUserRepository(db *sql.DB) repository.Repository {
	return &UserRepository{db: db}
}

// getExecutor returns the active executor (transaction if exists, otherwise db)
func (r *UserRepository) getExecutor() dbExecutor {
	if r.txn != nil {
		return r.txn
	}
	return r.db
}

// WithinTransaction executes a function within a database transaction
func (r *UserRepository) WithinTransaction(ctx context.Context, fn func(repo repository.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new repository instance with the transaction
	txRepo := &UserRepository{
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

// Create inserts a new user into the database.
func (r *UserRepository) Create(ctx context.Context, resource repository.Resource) (repository.Resource, error) {
	user, ok := resource.(*model.User)
	if !ok {
		return nil, errors.New("resource must be a *model.User")
	}

	user.InitMeta()

	query := `INSERT INTO users (id, email, password, name, region, status, role, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, user.ID, user.Email, user.Password, user.Name, user.Region, user.Status, user.Role, user.CreatedAt, user.UpdatedAt)
	if err != nil {
		var pqErr *pq.Error
		if errors.As(err, &pqErr) && pqErr.Code == pqUniqueViolationErrCode {
			return nil, &repository.UniqueConstraintError{Detail: pqErr.Detail}
		}
		return nil, fmt.Errorf("failed to insert user: %w", err)
	}

	return user, nil
}

// List retrieves users from the database based on the provided query.
func (r *UserRepository) List(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT * FROM users WHERE 1=1")

	var args []interface{}
	argIndex := 1

	// Apply query filters
	for field, value := range query.Values {
		switch field {
		case repository.IDField:
			queryBuilder.WriteString(fmt.Sprintf(" AND id = $%d", argIndex))
			id, err := uuid.Parse(value)
			if err != nil {
				return nil, fmt.Errorf("invalid ID format: %w", err)
			}
			args = append(args, id)
			argIndex++
		case repository.NameField:
			queryBuilder.WriteString(fmt.Sprintf(" AND name = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case repository.Region:
			queryBuilder.WriteString(fmt.Sprintf(" AND region = $%d", argIndex))
			args = append(args, value)
			argIndex++
		}
	}

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
		return nil, fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	var users []repository.Resource
	for rows.Next() {
		var user model.User
		err := rows.Scan(&user.ID, &user.Email, &user.Password, &user.Name, &user.Region, &user.Status, &user.Role, &user.CreatedAt, &user.UpdatedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan user: %w", err)
		}
		users = append(users, &user)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return users, nil
}

// FindByID retrieves a single user by ID.
func (r *UserRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.Resource, error) {
	query := `SELECT * FROM users WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	var result model.User
	err = stmt.QueryRowContext(ctx, id).Scan(
		&result.ID, &result.Email, &result.Password, &result.Name, &result.Region,
		&result.Status, &result.Role, &result.CreatedAt, &result.UpdatedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("user not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query user: %w", err)
	}

	return &result, nil
}

// DeleteByID deletes a user by ID.
func (r *UserRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM users WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("user not found")
	}

	return nil
}
