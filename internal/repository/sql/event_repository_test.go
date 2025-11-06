package sql

import (
	"context"
	"encoding/json"
	"errors"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		eventData := json.RawMessage(`{"product_id": "123", "action": "created"}`)
		event := &model.Event{
			EventType: "product.created",
			EventData: eventData,
			Status:    model.EventStatusPending,
		}

		mock.ExpectPrepare("INSERT INTO events").
			ExpectExec().
			WithArgs(sqlmock.AnyArg(), event.EventType, event.EventData, event.Status, sqlmock.AnyArg(), nil).
			WillReturnResult(sqlmock.NewResult(1, 1))

		result, err := repo.Create(ctx, event)
		require.NoError(t, err)
		assert.NotNil(t, result)

		createdEvent, ok := result.(*model.Event)
		require.True(t, ok)
		assert.NotEqual(t, uuid.Nil, createdEvent.ID)
		assert.Equal(t, "product.created", createdEvent.EventType)
		assert.Equal(t, eventData, createdEvent.EventData)
		assert.Equal(t, model.EventStatusPending, createdEvent.Status)
		assert.False(t, createdEvent.CreatedAt.IsZero())

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	t.Run("successful find", func(t *testing.T) {
		id := uuid.New()
		eventData := json.RawMessage(`{"product_id": "123"}`)
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}).
			AddRow(id, "product.created", eventData, model.EventStatusPending, createdAt, nil)

		mock.ExpectPrepare("SELECT \\* FROM events WHERE id").
			ExpectQuery().
			WithArgs(id).
			WillReturnRows(rows)

		result, err := repo.FindByID(ctx, id)
		require.NoError(t, err)
		assert.NotNil(t, result)

		event, ok := result.(*model.Event)
		require.True(t, ok)
		assert.Equal(t, id, event.ID)
		assert.Equal(t, "product.created", event.EventType)
		assert.Equal(t, eventData, event.EventData)
		assert.Equal(t, model.EventStatusPending, event.Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("event not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("SELECT \\* FROM events WHERE id").
			ExpectQuery().
			WithArgs(id).
			WillReturnRows(sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}))

		result, err := repo.FindByID(ctx, id)
		require.Error(t, err)
		assert.Nil(t, result)
		assert.Contains(t, err.Error(), "event not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	t.Run("list without filters", func(t *testing.T) {
		id1 := uuid.New()
		id2 := uuid.New()
		eventData1 := json.RawMessage(`{"product_id": "123"}`)
		eventData2 := json.RawMessage(`{"product_id": "456"}`)
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}).
			AddRow(id1, "product.created", eventData1, model.EventStatusPending, createdAt, nil).
			AddRow(id2, "product.deleted", eventData2, model.EventStatusProcessed, createdAt, &createdAt)

		mock.ExpectPrepare("SELECT \\* FROM events").
			ExpectQuery().
			WithArgs(10).
			WillReturnRows(rows)

		query := repository.NewQuery()
		results, err := repo.List(ctx, *query)
		require.NoError(t, err)
		assert.Len(t, results, 2)

		event1, ok := results[0].(*model.Event)
		require.True(t, ok)
		assert.Equal(t, id1, event1.ID)
		assert.Equal(t, "product.created", event1.EventType)

		event2, ok := results[1].(*model.Event)
		require.True(t, ok)
		assert.Equal(t, id2, event2.ID)
		assert.Equal(t, "product.deleted", event2.EventType)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_DeleteByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("DELETE FROM events WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := repo.DeleteByID(ctx, id)
		require.NoError(t, err)

		assert.NoError(t, mock.ExpectationsWereMet())
	})

	t.Run("event not found", func(t *testing.T) {
		id := uuid.New()

		mock.ExpectPrepare("DELETE FROM events WHERE id").
			ExpectExec().
			WithArgs(id).
			WillReturnResult(sqlmock.NewResult(0, 0))

		err := repo.DeleteByID(ctx, id)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "event not found")

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_WithinTransaction_Commit(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	eventData := json.RawMessage(`{"product_id": "123"}`)
	event := &model.Event{
		EventType: "product.created",
		EventData: eventData,
		Status:    model.EventStatusPending,
	}

	mock.ExpectBegin()
	mock.ExpectPrepare("INSERT INTO events").
		ExpectExec().
		WithArgs(sqlmock.AnyArg(), event.EventType, event.EventData, event.Status, sqlmock.AnyArg(), nil).
		WillReturnResult(sqlmock.NewResult(1, 1))
	mock.ExpectCommit()

	err = repo.WithinTransaction(ctx, func(txRepo repository.Repository) error {
		_, err := txRepo.Create(ctx, event)
		return err
	})

	require.NoError(t, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}

func TestEventRepository_WithinTransaction_Rollback(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := NewEventRepository(db)
	ctx := context.Background()

	expectedErr := errors.New("transaction error")

	mock.ExpectBegin()
	mock.ExpectRollback()

	err = repo.WithinTransaction(ctx, func(_ repository.Repository) error {
		return expectedErr
	})

	require.Error(t, err)
	assert.Equal(t, expectedErr, err)
	assert.NoError(t, mock.ExpectationsWereMet())
}
