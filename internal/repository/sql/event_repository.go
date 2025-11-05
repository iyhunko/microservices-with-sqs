package sql

import (
	"context"
	"database/sql"
	"encoding/json"
	"errors"
	"fmt"

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
func NewEventRepository(db *sql.DB) repository.Repository {
	return &EventRepository{db: db}
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

// FindByID retrieves a single event by ID.
func (r *EventRepository) FindByID(ctx context.Context, id uuid.UUID) (repository.Resource, error) {
	query := `SELECT id, event_type, event_data, status, created_at, processed_at FROM events WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	var result model.Event
	var processedAt sql.NullTime
	err = stmt.QueryRowContext(ctx, id).Scan(
		&result.ID, &result.EventType, &result.EventData, &result.Status, &result.CreatedAt, &processedAt,
	)
	if err != nil {
		if err == sql.ErrNoRows {
			return nil, fmt.Errorf("event not found: %w", err)
		}
		return nil, fmt.Errorf("failed to query event: %w", err)
	}

	if processedAt.Valid {
		result.ProcessedAt = &processedAt.Time
	}

	return &result, nil
}

// List retrieves events from the database based on the provided query.
func (r *EventRepository) List(ctx context.Context, query repository.Query) ([]repository.Resource, error) {
	sqlQuery := `SELECT id, event_type, event_data, status, created_at, processed_at 
	             FROM events 
	             WHERE status = $1 
	             ORDER BY created_at ASC 
	             LIMIT $2`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, sqlQuery)
	if err != nil {
		return nil, fmt.Errorf("failed to prepare select statement: %w", err)
	}
	defer stmt.Close()

	limit := query.Limit
	if limit <= 0 {
		limit = repository.DefaultPaginationLimit
	}

	rows, err := stmt.QueryContext(ctx, model.EventStatusPending, limit)
	if err != nil {
		return nil, fmt.Errorf("failed to query events: %w", err)
	}
	defer rows.Close()

	var events []repository.Resource
	for rows.Next() {
		var event model.Event
		var processedAt sql.NullTime
		err := rows.Scan(&event.ID, &event.EventType, &event.EventData, &event.Status, &event.CreatedAt, &processedAt)
		if err != nil {
			return nil, fmt.Errorf("failed to scan event: %w", err)
		}
		if processedAt.Valid {
			event.ProcessedAt = &processedAt.Time
		}
		events = append(events, &event)
	}

	if err = rows.Err(); err != nil {
		return nil, fmt.Errorf("error iterating rows: %w", err)
	}

	return events, nil
}

// DeleteByID deletes an event by ID.
func (r *EventRepository) DeleteByID(ctx context.Context, resource repository.Resource) error {
	event, ok := resource.(*model.Event)
	if !ok {
		return errors.New("resource must be a *model.Event")
	}

	query := `DELETE FROM events WHERE id = $1`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare delete statement: %w", err)
	}
	defer stmt.Close()

	result, err := stmt.ExecContext(ctx, event.ID)
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

// UpdateStatus updates the status and processed_at time of an event
func (r *EventRepository) UpdateStatus(ctx context.Context, eventID uuid.UUID, status interface{}) error {
	eventStatus, ok := status.(model.EventStatus)
	if !ok {
		return fmt.Errorf("status must be of type model.EventStatus")
	}

	query := `UPDATE events SET status = $1, processed_at = CURRENT_TIMESTAMP WHERE id = $2`

	executor := r.getExecutor()
	stmt, err := executor.PrepareContext(ctx, query)
	if err != nil {
		return fmt.Errorf("failed to prepare update statement: %w", err)
	}
	defer stmt.Close()

	_, err = stmt.ExecContext(ctx, eventStatus, eventID)
	if err != nil {
		return fmt.Errorf("failed to update event status: %w", err)
	}

	return nil
}

// CreateEvent is a helper function to create an event with proper JSON marshaling
func CreateEvent(eventType string, eventData interface{}) (*model.Event, error) {
	data, err := json.Marshal(eventData)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal event data: %w", err)
	}

	return &model.Event{
		EventType: eventType,
		EventData: data,
		Status:    model.EventStatusPending,
	}, nil
}
