package model

import (
	"encoding/json"
	"time"

	"github.com/google/uuid"
)

// EventStatus represents the status of an event in the outbox pattern.
type EventStatus string

const (
	// EventStatusPending indicates the event has been created but not yet processed
	EventStatusPending EventStatus = "pending"
	// EventStatusProcessed indicates the event has been successfully processed
	EventStatusProcessed EventStatus = "processed"
	// EventStatusFailed indicates the event processing has failed
	EventStatusFailed EventStatus = "failed"
)

// Event represents an event entity for the outbox pattern.
type Event struct {
	ID          uuid.UUID
	EventType   string
	EventData   json.RawMessage
	Status      EventStatus
	CreatedAt   time.Time
	ProcessedAt *time.Time
}

// InitMeta initializes the event metadata including ID and timestamps.
func (e *Event) InitMeta() {
	e.ID = uuid.New()
	e.CreatedAt = time.Now()
	if e.Status == "" {
		e.Status = EventStatusPending
	}
}
