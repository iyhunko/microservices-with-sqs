package model

import (
	"time"

	"github.com/google/uuid"
)

type User struct {
	ID        uuid.UUID `gorm:"column:id;type:uuid;primaryKey"`
	Email     string    `gorm:"column:email;uniqueIndex"`
	Password  string    `gorm:"column:password"`
	Name      string    `gorm:"column:name"`
	Region    string    `gorm:"column:region"`
	Status    string    `gorm:"column:status"`
	Role      string    `gorm:"column:role"`
	UpdatedAt time.Time `gorm:"column:updated_at;autoUpdateTime"`
	CreatedAt time.Time `gorm:"column:created_at;autoCreateTime"`
}

func (t *User) TableName() string {
	return "users"
}

func (t *User) InitMeta() {
	t.ID = uuid.New()
}
