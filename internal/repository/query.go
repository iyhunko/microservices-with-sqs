package repository

import (
	"errors"
	"log/slog"
)

const (
	NotEmpty QueryFieldValue = "not_empty"
	Empty    QueryFieldValue = "empty"

	IDField        QueryField = "id"
	NameField      QueryField = "name"
	Region         QueryField = "region"
	CreatedAtField QueryField = "created_at"
)

type Query struct {
	Values map[QueryField]string

	Limit int

	Paginator *Paginator
}

type QueryField string

type QueryFieldValue string

func NewQuery() *Query {
	return &Query{
		Values: map[QueryField]string{},
	}
}

func (q *Query) With(field QueryField, val string) *Query {
	q.Values[field] = val
	return q
}

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
