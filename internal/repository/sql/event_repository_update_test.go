package sql

import (
	"context"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db).(*EventRepository)
	ctx := context.Background()

	t.Run("successful status update to processed", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("UPDATE events SET status").
			ExpectExec().
			WithArgs(model.EventStatusProcessed, sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateStatus(ctx, id, model.EventStatusProcessed)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("successful status update to failed", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("UPDATE events SET status").
			ExpectExec().
			WithArgs(model.EventStatusFailed, sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.UpdateStatus(ctx, id, model.EventStatusFailed)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("event not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("UPDATE events SET status").
			ExpectExec().
			WithArgs(model.EventStatusProcessed, sqlmock.AnyArg(), id).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.UpdateStatus(ctx, id, model.EventStatusProcessed)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_ListWithStatusFilter(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	t.Run("list with pending status filter", func(t *testing.T) {
		id := uuid.New()
		eventData := []byte(`{"product_id": "123"}`)
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}).
			AddRow(id, "product.created", eventData, model.EventStatusPending, createdAt, nil)

		mock.ExpectPrepare("SELECT \\* FROM events").
			ExpectQuery().
			WithArgs(string(model.EventStatusPending), 50).
			WillReturnRows(rows)

		query := repository.NewQuery().With(repository.StatusField, string(model.EventStatusPending))
		results, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, results, 1)

		event, ok := results[0].(*model.Event)
		require.True(t, ok)
		assert.Equal(t, model.EventStatusPending, event.Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
