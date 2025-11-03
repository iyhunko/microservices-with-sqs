package repository

import "context"

type TransactionFunc func(context.Context, Repository) error

// Repository defines the interface for a generic repository that can manage resources.
type Repository interface {
	Create(ctx context.Context, resource Resource) error
	List(ctx context.Context, result any, query Query) error
	Delete(ctx context.Context, resource Resource) error
	Find(ctx context.Context, resource Resource) (bool, error)
	Patch(ctx context.Context, resource Resource) (bool, error)
	Transaction(ctx context.Context, txFunc TransactionFunc) error
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
