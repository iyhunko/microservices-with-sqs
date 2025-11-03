package sql

import (
	"context"
	"errors"
	"fmt"
	"log/slog"

	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/jackc/pgx/v5/pgconn"
	"gorm.io/gorm"
)

const (
	pqUniqueViolationErrCode = "23505" // PostgreSQL unique violation error code. See https://www.postgresql.org/docs/14/errcodes-appendix.html
)

type ResourceRepository struct {
	db *gorm.DB
}

func NewRepository(db *gorm.DB) *ResourceRepository {
	return &ResourceRepository{db: db}
}

func (r ResourceRepository) Create(ctx context.Context, resource repository.Resource) error {
	resource.InitMeta()
	if err := r.db.WithContext(ctx).Create(resource).Error; err != nil {
		slog.Error("error creating resource", slog.Any("err", err))
		var pgError *pgconn.PgError
		if errors.As(err, &pgError) && pgError.Code == pqUniqueViolationErrCode {
			return &repository.UniqueConstraintError{Detail: pgError.Detail}
		}
		return fmt.Errorf("failed to create resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) List(ctx context.Context, result any, query repository.Query) error {
	dbQuery := r.db.WithContext(ctx).Model(result)
	for field, value := range query.Values {
		switch value {
		case string(repository.NotEmpty):
			dbQuery = dbQuery.Where(field + " IS NOT NULL AND " + field + " != ''")
		case string(repository.Empty):
			dbQuery = dbQuery.Where(field + " IS NULL OR " + field + " = ''")
		default:
			dbQuery = dbQuery.Where(field+" = ?", value)
		}
	}

	if query.Paginator != nil {
		dbQuery.Where(string(repository.IDField)+" != ?", query.Paginator.LastID)
		dbQuery.Where(string(repository.CreatedAtField)+" <= ?", query.Paginator.LastCreatedAt)
	}

	if query.Limit == 0 {
		query.Limit = repository.DefaultPaginationLimit
	}
	dbQuery.Limit(query.Limit)
	dbQuery.Order(fmt.Sprintf("%s desc, %s", repository.CreatedAtField, repository.IDField))

	if err := dbQuery.Find(result).Error; err != nil {
		return fmt.Errorf("failed to list resources: %w", err)
	}

	return nil
}

func (r ResourceRepository) Delete(ctx context.Context, resource repository.Resource) error {
	if err := r.db.WithContext(ctx).Delete(resource).Error; err != nil {
		slog.Error("error deleting resource", slog.Any("err", err))
		return fmt.Errorf("failed to delete resource: %w", err)
	}
	return nil
}

func (r ResourceRepository) Find(ctx context.Context, resource repository.Resource) (bool, error) {
	result := r.db.WithContext(ctx).Limit(1).Find(resource)
	if result.Error != nil {
		slog.Error("error finding a resource", slog.Any("err", result.Error))
		return false, fmt.Errorf("failed to find resource: %w", result.Error)
	}

	return result.RowsAffected > 0, nil
}

func (r ResourceRepository) Patch(ctx context.Context, resource repository.Resource) (bool, error) {
	db := r.db.WithContext(ctx).Model(resource).Updates(resource)
	if err := db.Error; err != nil {
		slog.Error("error patching resource", slog.Any("err", err))
		return false, fmt.Errorf("failed to patch resource: %w", err)
	}

	return db.RowsAffected > 0, nil
}
