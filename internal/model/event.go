package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the status of an event in the outbox pattern.
type EventStatus string

const (
	// EventStatusPending indicates the event has been created but not yet processed.
	EventStatusPending EventStatus = "pending"
	// EventStatusProcessed indicates the event has been successfully processed.
	EventStatusProcessed EventStatus = "processed"
	// EventStatusFailed indicates the event processing has failed.
	EventStatusFailed EventStatus = "failed"
)

// Event represents an event entity for the outbox pattern.
type Event struct {
	ID          uuid.UUID       `db:"id"`
	EventType   string          `db:"event_type"`
	EventData   json.RawMessage `db:"event_data"`
	Status      EventStatus     `db:"status"`
	CreatedAt   time.Time       `db:"created_at"`
	ProcessedAt *time.Time      `db:"processed_at"`
}

// TableName returns the database table name for the Event model.
func (e *Event) TableName() string {
	return "events"
}

// InitMeta initializes the event metadata including ID and timestamps.
func (e *Event) InitMeta() {
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	if e.Status == "" {
		e.Status = EventStatusPending
	}
}
