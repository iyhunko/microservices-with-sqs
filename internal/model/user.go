package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"column:id;primaryKey" db:"id"`
	Email     string    `gorm:"column:email" db:"email"`
	Password  string    `gorm:"column:password" db:"password"`
	Name      string    `gorm:"column:name" db:"name"`
	Region    string    `gorm:"column:region" db:"region"`
	Status    string    `gorm:"column:status" db:"status"`
	Role      string    `gorm:"column:role" db:"role"`
	UpdatedAt time.Time `gorm:"column:updated_at" db:"updated_at"`
	CreatedAt time.Time `gorm:"column:created_at" db:"created_at"`
}

func (t *User) TableName() string {
	return "users"
}

func (t *User) InitMeta() {
	t.ID = uuid.New()
	t.CreatedAt = time.Now()
	t.UpdatedAt = time.Now()
}
