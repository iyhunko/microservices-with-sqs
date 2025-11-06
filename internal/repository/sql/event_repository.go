package sql

import (
	"context"
	"database/sql"
	"errors"
	"fmt"
	"strings"
	"time"

	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
)

// EventRepository implements the Repository interface for Event entities.
type EventRepository struct {
	db  *sql.DB
	txn *sql.Tx
}

// NewEventRepository creates a new EventRepository instance.
func NewEventRepository(db *sql.DB) *EventRepository {
	return &EventRepository{db: db}
}

// NewEventRepositoryWithTx creates a new EventRepository instance with an existing transaction.
func NewEventRepositoryWithTx(db *sql.DB, tx *sql.Tx) *EventRepository {
	return &EventRepository{db: db, txn: tx}
}

// getExecutor returns the active executor (transaction if exists, otherwise db)
func (r *EventRepository) getExecutor() dbExecutor {
	if r.txn != nil {
		return r.txn
	}
	return r.db
}

// WithinTransaction executes a function within a database transaction
func (r *EventRepository) WithinTransaction(ctx context.Context, fn func(repo repository.Repository) error) error {
	tx, err := r.db.BeginTx(ctx, nil)
	if err != nil {
		return fmt.Errorf("failed to begin transaction: %w", err)
	}

	// Create a new repository instance with the transaction
	txRepo := &EventRepository{
		db:  r.db,
		txn: tx,
	}

	// Execute the function with the transactional repository
	if err := fn(txRepo); err != nil {
		if rbErr := tx.Rollback(); rbErr != nil {
			// Log rollback error but return original error
			return fmt.Errorf("transaction failed (rollback error: %v): %w", rbErr, err)
		}
		return err
	}

	// Commit the transaction
	if err := tx.Commit(); err != nil {
		return fmt.Errorf("failed to commit transaction: %w", err)
	}

	return nil
}

// Create inserts a new event into the database.
func (r *EventRepository) Create(ctx context.Context, resource repository.Resource) (repository.Resource, error) {
	event, ok := resource.(*model.Event)
	if !ok {
		return nil, errors.New("resource must be a *model.Event")
	}

	event.InitMeta()

	query := `INSERT INTO events (id, event_type, event_data, status, created_at, processed_at) 
	          VALUES ($1, $2, $3, $4, $5, $6)`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare insert statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, event.ID, event.EventType, event.EventData, event.Status, event.CreatedAt, event.ProcessedAt)
	if err != nil {
		return nil, fmt.Errorf("failed to insert event: %w", err)
	}

	return event, nil
}

// List retrieves events from the database based on the provided query.
func (r *EventRepository) List(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	var queryBuilder strings.Builder
	queryBuilder.WriteString("SELECT * FROM events WHERE 1=1")

	var args []interface{}
	argIndex := 1

	// Apply status filter if provided
	if status, ok := query.Values[repository.StatusField]; ok {
		queryBuilder.WriteString(fmt.Sprintf(" AND status = $%d", argIndex))
		args = append(args, status)
		argIndex++
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
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []repository.Resource
	for rows.Next() {
		var event model.Event
		err := rows.Scan(&event.ID, &event.EventType, &event.EventData, &event.Status, &event.CreatedAt, &event.ProcessedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}

// FindByID retrieves a single event by ID.
func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.Resource, error) {
	query := `SELECT * FROM events WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	var result model.Event
	err = stmt.QueryRowContext(ctx, id).Scan(
		&result.ID, &result.EventType, &result.EventData, &result.Status, &result.CreatedAt, &result.ProcessedAt,
	)
	if err != nil {
		if errors.Is(err, sql.ErrNoRows) {
			return nil, fmt.Errorf("event not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query event: %w", err)
	}

	return &result, nil
}

// DeleteByID deletes an event by ID.
func (r *EventRepository) DeleteByID(ctx context.Context, id uuid.UUID) error {
	query := `DELETE FROM events WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, id)
	if err != nil {
		return fmt.Errorf("failed to delete event: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}

// UpdateStatus updates the status of an event by ID.
func (r *EventRepository) UpdateStatus(ctx context.Context, id uuid.UUID, status model.EventStatus) error {
	query := `UPDATE events SET status = $1, processed_at = $2 WHERE id = $3`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	var processedAt interface{}
	if status == model.EventStatusProcessed || status == model.EventStatusFailed {
		now := time.Now()
		processedAt = &now
	}

	result, err := stmt.ExecContext(ctx, status, processedAt, id)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	rowsAffected, err := result.RowsAffected()
	if err != nil {
		return fmt.Errorf("failed to get rows affected: %w", err)
	}

	if rowsAffected == 0 {
		return fmt.Errorf("event not found")
	}

	return nil
}
