package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user entity with authentication and profile information.
type User struct {
	ID        uuid.UUID
	Email     string
	Password  string
	Name      string
	Region    string
	Status    string
	Role      string
	UpdatedAt time.Time
	CreatedAt time.Time
}

// InitMeta initializes the user metadata including ID and timestamps.
func (t *User) InitMeta() {
	t.ID = uuid.New()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
}
