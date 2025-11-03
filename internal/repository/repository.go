package repository

import "context"

// Repository defines the interface for a generic repository that can manage resources.
type Repository interface {
	Create(ctx context.Context, resource Resource) (result Resource, err error)
	List(ctx context.Context, query Query) (result []Resource, err error)
	DeleteByID(ctx context.Context, resource Resource) error
	FindByID(ctx context.Context, resource Resource) (result Resource, err error) // find one
}

// Resource represents a generic resource that can be managed by the repository.
type Resource interface {
	InitMeta()
}

// UniqueConstraintError represents a database unique constraint violation error.
type UniqueConstraintError struct {
	Detail string
}

// NewUniqueConstraintError creates a new UniqueConstraintError with the provided detail message.
func (u *UniqueConstraintError) Error() string {
	return "resource must be unique: " + u.Detail
}
