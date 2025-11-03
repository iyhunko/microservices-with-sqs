package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"

	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/lib/pq"
)

// UserRepository implements repository.Repository for User model using raw SQL
type UserRepository struct {
	db *sql.DB
}

// NewRepository creates a new UserRepository instance
func NewRepository(db *sql.DB) repository.Repository {
	return &UserRepository{db: db}
}

// Create inserts a new user into the database
func (r *UserRepository) Create(ctx context.Context, resource repository.Resource) error {
	user, ok := resource.(*model.User)
	if !ok {
		return errors.New("resource must be of type *model.User")
	}

	query := `
		INSERT INTO users (id, email, password, name, region, status, role, created_at, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx,
		user.ID,
		user.Email,
		user.Password,
		user.Name,
		user.Region,
		user.Status,
		user.Role,
		user.CreatedAt,
		user.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return &repository.UniqueConstraintError{Detail: pqErr.Detail}
			}
		}
		return fmt.Errorf("failed to insert user: %w", err)
	}

	return nil
}

// List retrieves users from the database based on the query
func (r *UserRepository) List(ctx context.Context, result any, query repository.Query) error {
	users, ok := result.(*[]*model.User)
	if !ok {
		return errors.New("result must be of type *[]*model.User")
	}

	var args []any
	argIndex := 1

	sqlQuery := "SELECT id, email, password, name, region, status, role, created_at, updated_at FROM users WHERE 1=1"
	var whereClauses []string

	// Apply filters from query
	for field, value := range query.Values {
		switch field {
		case repository.IDField:
			whereClauses = append(whereClauses, fmt.Sprintf("id = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case repository.NameField:
			whereClauses = append(whereClauses, fmt.Sprintf("name = $%d", argIndex))
			args = append(args, value)
			argIndex++
		case repository.Region:
			whereClauses = append(whereClauses, fmt.Sprintf("region = $%d", argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	if len(whereClauses) > 0 {
		sqlQuery += " AND " + strings.Join(whereClauses, " AND ")
	}

	// Apply pagination
	if query.Paginator != nil {
		sqlQuery += fmt.Sprintf(" AND (created_at, id) > ($%d, $%d)", argIndex, argIndex+1)
		args = append(args, query.Paginator.LastCreatedAt, query.Paginator.LastID)
		argIndex += 2
	}

	// Order by created_at and id for consistent pagination
	sqlQuery += " ORDER BY created_at, id"

	// Apply limit
	if query.Limit > 0 {
		sqlQuery += fmt.Sprintf(" LIMIT $%d", argIndex)
		args = append(args, query.Limit)
	}

	stmt, err := r.db.PrepareContext(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to query users: %w", err)
	}
	defer rows.Close()

	for rows.Next() {
		user := &model.User{}
		err := rows.Scan(
			&user.ID,
			&user.Email,
			&user.Password,
			&user.Name,
			&user.Region,
			&user.Status,
			&user.Role,
			&user.CreatedAt,
			&user.UpdatedAt,
		)
		if err != nil {
			return fmt.Errorf("failed to scan user: %w", err)
		}
		*users = append(*users, user)
	}

	if err := rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	return nil
}

// Find retrieves a single user by ID
func (r *UserRepository) Find(ctx context.Context, resource repository.Resource) (bool, error) {
	user, ok := resource.(*model.User)
	if !ok {
		return false, errors.New("resource must be of type *model.User")
	}

	query := `
		SELECT id, email, password, name, region, status, role, created_at, updated_at
		FROM users
		WHERE id = $1
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	err = stmt.QueryRowContext(ctx, user.ID).Scan(
		&user.ID,
		&user.Email,
		&user.Password,
		&user.Name,
		&user.Region,
		&user.Status,
		&user.Role,
		&user.CreatedAt,
		&user.UpdatedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return false, nil
		}
		return false, fmt.Errorf("failed to find user: %w", err)
	}

	return true, nil
}

// Patch updates a user in the database
func (r *UserRepository) Patch(ctx context.Context, resource repository.Resource) (bool, error) {
	user, ok := resource.(*model.User)
	if !ok {
		return false, errors.New("resource must be of type *model.User")
	}

	query := `
		UPDATE users
		SET email = $2, password = $3, name = $4, region = $5, status = $6, role = $7, updated_at = $8
		WHERE id = $1
	`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx,
		user.ID,
		user.Email,
		user.Password,
		user.Name,
		user.Region,
		user.Status,
		user.Role,
		user.UpdatedAt,
	)
	if err != nil {
		if pqErr, ok := err.(*pq.Error); ok {
			if pqErr.Code == "23505" { // unique_violation
				return false, &repository.UniqueConstraintError{Detail: pqErr.Detail}
			}
		}
		return false, fmt.Errorf("failed to update user: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

// Delete removes a user from the database by ID
func (r *UserRepository) Delete(ctx context.Context, resource repository.Resource) error {
	user, ok := resource.(*model.User)
	if !ok {
		return errors.New("resource must be of type *model.User")
	}

	query := `DELETE FROM users WHERE id = $1`

	stmt, err := r.db.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, user.ID)
	if err != nil {
		return fmt.Errorf("failed to delete user: %w", err)
	}

	return nil
}
