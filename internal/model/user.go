package model

import (
	"time"

	"github.com/google/uuid"
)

// User represents a user entity with authentication and profile information.
type User struct {
	ID        uuid.UUID `db:"id"`
	Email     string    `db:"email"`
	Password  string    `db:"password"`
	Name      string    `db:"name"`
	Region    string    `db:"region"`
	Status    string    `db:"status"`
	Role      string    `db:"role"`
	UpdatedAt time.Time `db:"updated_at"`
	CreatedAt time.Time `db:"created_at"`
}

// TableName returns the database table name for the User model.
func (u *User) TableName() string {
	return "users"
}

// InitMeta initializes the user metadata including ID and timestamps.
func (t *User) InitMeta() {
	t.ID = uuid.New()
	now := time.Now()
	t.CreatedAt = now
	t.UpdatedAt = now
}
