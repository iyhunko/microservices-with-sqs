package model

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product entity with its properties and metadata.
type Product struct {
	ID          uuid.UUID
	Name        string
	Description string
	Price       float64
	UpdatedAt   time.Time
	CreatedAt   time.Time
}

// InitMeta initializes the product metadata including ID and timestamps.
func (p *Product) InitMeta() {
	p.ID = uuid.New()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
}
