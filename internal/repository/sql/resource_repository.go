package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"log/slog"
	"strings"

	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
)

const (
	pqUniqueViolationErrCode = "23505" // PostgreSQL unique violation error code. See https://www.postgresql.org/docs/14/errcodes-appendix.html
)

// allowedFields is a whitelist of valid column names to prevent SQL injection
var allowedFields = map[string]bool{
	"id":         true,
	"email":      true,
	"password":   true,
	"name":       true,
	"region":     true,
	"status":     true,
	"role":       true,
	"created_at": true,
	"updated_at": true,
}

// validateField checks if a field name is in the whitelist
func validateField(field string) error {
	if !allowedFields[field] {
		return fmt.Errorf("invalid field name: %s", field)
	}
	return nil
}

// dbExecutor is an interface that both sql.DB and sql.Tx implement
type dbExecutor interface {
	PrepareContext(ctx context.Context, query string) (*sql.Stmt, error)
	ExecContext(ctx context.Context, query string, args ...interface{}) (sql.Result, error)
	QueryContext(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error)
	QueryRowContext(ctx context.Context, query string, args ...interface{}) *sql.Row
}

type ResourceRepository struct {
	db  *sql.DB
	tx  *sql.Tx
	exe dbExecutor
}

func NewRepository(db *sql.DB) *ResourceRepository {
	return &ResourceRepository{db: db, exe: db}
}

func newRepositoryWithTx(tx *sql.Tx) *ResourceRepository {
	return &ResourceRepository{tx: tx, exe: tx}
}

func (r ResourceRepository) Create(ctx context.Context, resource repository.Resource) error {
	resource.InitMeta()

	// Note: This implementation is intentionally type-specific to avoid using reflection,
	// which is safer for production code. If additional resource types are needed,
	// they should be added as explicit type assertions with their own SQL queries.
	user, ok := resource.(*model.User)
	if !ok {
		return fmt.Errorf("unsupported resource type")
	}

	query := `INSERT INTO users (id, email, password, name, region, status, role, created_at, updated_at) 
	          VALUES ($1, $2, $3, $4, $5, $6, $7, $8, $9)`

	stmt, err := r.exe.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare create statement: %w", err)
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
		slog.Error("error creating resource", slog.Any("err", err))
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pqUniqueViolationErrCode {
			return &repository.UniqueConstraintError{Detail: pgError.Detail}
		}
		return fmt.Errorf("failed to create resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) List(ctx context.Context, result any, query repository.Query) error {
	users, ok := result.(*[]*model.User)
	if !ok {
		return fmt.Errorf("unsupported result type, expected *[]*model.User")
	}

	// Build the WHERE clause
	var whereClauses []string
	var args []interface{}
	argIndex := 1

	for field, value := range query.Values {
		// Validate field name to prevent SQL injection
		if err := validateField(string(field)); err != nil {
			return fmt.Errorf("invalid query field: %w", err)
		}

		switch value {
		case string(repository.NotEmpty):
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NOT NULL AND %s != ''", field, field))
		case string(repository.Empty):
			whereClauses = append(whereClauses, fmt.Sprintf("%s IS NULL OR %s = ''", field, field))
		default:
			whereClauses = append(whereClauses, fmt.Sprintf("%s = $%d", field, argIndex))
			args = append(args, value)
			argIndex++
		}
	}

	if query.Paginator != nil {
		whereClauses = append(whereClauses, fmt.Sprintf("%s != $%d", repository.IDField, argIndex))
		args = append(args, query.Paginator.LastID)
		argIndex++
		whereClauses = append(whereClauses, fmt.Sprintf("%s <= $%d", repository.CreatedAtField, argIndex))
		args = append(args, query.Paginator.LastCreatedAt)
		argIndex++
	}

	if query.Limit == 0 {
		query.Limit = repository.DefaultPaginationLimit
	}

	sqlQuery := "SELECT id, email, password, name, region, status, role, created_at, updated_at FROM users"
	if len(whereClauses) > 0 {
		sqlQuery += " WHERE " + strings.Join(whereClauses, " AND ")
	}
	sqlQuery += fmt.Sprintf(" ORDER BY %s DESC, %s LIMIT $%d", repository.CreatedAtField, repository.IDField, argIndex)
	args = append(args, query.Limit)

	stmt, err := r.exe.PrepareContext(ctx, sqlQuery)
	if err != nil {
		return fmt.Errorf("failed to prepare list statement: %w", err)
	}
	defer stmt.Close()

	rows, err := stmt.QueryContext(ctx, args...)
	if err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}
	defer rows.Close()

	*users = []*model.User{}
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

	if err = rows.Err(); err != nil {
		return fmt.Errorf("error iterating rows: %w", err)
	}

	return nil
}

func (r ResourceRepository) Delete(ctx context.Context, resource repository.Resource) error {
	user, ok := resource.(*model.User)
	if !ok {
		return fmt.Errorf("unsupported resource type")
	}

	query := "DELETE FROM users WHERE id = $1"
	stmt, err := r.exe.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, user.ID)
	if err != nil {
		slog.Error("error deleting resource", slog.Any("err", err))
		return fmt.Errorf("failed to delete resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) Find(ctx context.Context, resource repository.Resource) (bool, error) {
	user, ok := resource.(*model.User)
	if !ok {
		return false, fmt.Errorf("unsupported resource type")
	}

	query := "SELECT id, email, password, name, region, status, role, created_at, updated_at FROM users WHERE id = $1 LIMIT 1"
	stmt, err := r.exe.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare find statement: %w", err)
	}
	defer stmt.Close()

	row := stmt.QueryRowContext(ctx, user.ID)
	err = row.Scan(
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

	if err == sql.ErrNoRows {
		return false, nil
	}

	if err != nil {
		slog.Error("error finding a resource", slog.Any("err", err))
		return false, fmt.Errorf("failed to find resource: %w", err)
	}

	return true, nil
}

func (r ResourceRepository) Patch(ctx context.Context, resource repository.Resource) (bool, error) {
	user, ok := resource.(*model.User)
	if !ok {
		return false, fmt.Errorf("unsupported resource type")
	}

	query := `UPDATE users 
	          SET email = $1, password = $2, name = $3, region = $4, status = $5, role = $6, updated_at = $7
	          WHERE id = $8`

	stmt, err := r.exe.PrepareContext(ctx, query)
	if err != nil {
		return false, fmt.Errorf("failed to prepare patch statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx,
		user.Email,
		user.Password,
		user.Name,
		user.Region,
		user.Status,
		user.Role,
		user.UpdatedAt,
		user.ID,
	)

	if err != nil {
		slog.Error("error patching resource", slog.Any("err", err))
		return false, fmt.Errorf("failed to patch resource: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return false, fmt.Errorf("failed to get rows affected: %w", err)
	}

	return rowsAffected > 0, nil
}

func (r ResourceRepository) Transaction(ctx context.Context, txFunc repository.TransactionFunc) error {
	// If we're already in a transaction, just execute the function
	if r.tx != nil {
		return txFunc(ctx, r)
	}

	// Start a new transaction
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new repository using the transaction
	txRepo := newRepositoryWithTx(tx)

	defer func() {
		if p := recover(); p != nil {
			_ = tx.Rollback()
			panic(p)
		}
	}()

	// Execute the transaction function with the transactional repository
	if err := txFunc(ctx, *txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			return fmt.Errorf("failed to rollback transaction: %w (original error: %v)", rbErr, err)
		}
		return err
	}

	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}
