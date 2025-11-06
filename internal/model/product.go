package model

import (
	"time"

	"github.com/google/uuid"
)

// Product represents a product entity with its properties and metadata.
type Product struct {
	ID          uuid.UUID `db:"id"`
	Name        string    `db:"name"`
	Description string    `db:"description"`
	Price       float64   `db:"price"`
	UpdatedAt   time.Time `db:"updated_at"`
	CreatedAt   time.Time `db:"created_at"`
}

// TableName returns the database table name for the Product model.
func (p *Product) TableName() string {
	return "products"
}

// InitMeta initializes the product metadata including ID and timestamps.
func (p *Product) InitMeta() {
	p.ID = uuid.New()
	now := time.Now()
	p.CreatedAt = now
	p.UpdatedAt = now
}
