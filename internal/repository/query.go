package repository

import (
	"errors"
	"log/slog"
)

const (
	// NotEmpty represents a non-empty query field value.
	NotEmpty QueryFieldValue = "not_empty"
	// Empty represents an empty query field value.
	Empty QueryFieldValue = "empty"

	// IDField represents the ID query field.
	IDField QueryField = "id"
	// NameField represents the name query field.
	NameField QueryField = "name"
	// Region represents the region query field.
	Region QueryField = "region"
	// CreatedAtField represents the created_at query field.
	CreatedAtField QueryField = "created_at"
)

// Query represents a database query with filters and pagination options.
type Query struct {
	Values map[QueryField]string

	Limit int

	Paginator *Paginator
}

// QueryField represents a field name used in database queries.
type QueryField string

// QueryFieldValue represents a value for a query field.
type QueryFieldValue string

// NewQuery creates a new Query instance with an empty values map.
func NewQuery() *Query {
	return &Query{
		Values: map[QueryField]string{},
	}
}

// With adds a field filter to the query and returns the query for chaining.
func (q *Query) With(field QueryField, val string) *Query {
	q.Values[field] = val
	return q
}

// ApplyPagination applies pagination settings to the query based on limit and page token.
func (q *Query) ApplyPagination(limit int32, token string) error {
	queryLimit := DefaultPaginationLimit
	if limit > 0 {
		queryLimit = min(maxPaginationLimit, int(limit))
	}
	q.Limit = queryLimit

	if token == "" {
		return nil
	}

	paginator, err := DecodePageToken(token)
	if err != nil {
		slog.Error("failed to decode page token", slog.Any("err", err), slog.String("token", token))
		return errors.New("invalid page token")
	}
	q.Paginator = paginator
	return nil
}
