package sql_test

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/DATA-DOG/go-sqlmock"
	"github.com/google/uuid"
	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEventRepository_Create(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := sql.NewEventRepository(db)
	ctx := context.Background()

	t.Run("successful creation", func(t *testing.T) {
		eventData := map[string]interface{}{
			"action":     "created",
			"product_id": uuid.New().String(),
			"name":       "Test Product",
			"price":      99.99,
		}
		eventDataJSON, err := json.Marshal(eventData)
		require.NoError(t, err)

		event := &model.Event{
			EventType: "product.created",
			EventData: eventDataJSON,
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
		assert.Equal(t, model.EventStatusPending, createdEvent.Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_FindByID(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := sql.NewEventRepository(db)
	ctx := context.Background()

	t.Run("successful find", func(t *testing.T) {
		eventID := uuid.New()
		eventData := json.RawMessage(`{"action":"created"}`)
		createdAt := time.Now()

		rows := sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}).
			AddRow(eventID, "product.created", eventData, "pending", createdAt, nil)

		mock.ExpectPrepare("SELECT (.+) FROM events WHERE id").
			ExpectQuery().
			WithArgs(eventID).
			WillReturnRows(rows)

		result, err := repo.FindByID(ctx, eventID)

		require.NoError(t, err)
		assert.NotNil(t, result)

		event, ok := result.(*model.Event)
		require.True(t, ok)
		assert.Equal(t, eventID, event.ID)
		assert.Equal(t, "product.created", event.EventType)
		assert.Equal(t, model.EventStatusPending, event.Status)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_List(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	repo := sql.NewEventRepository(db)
	ctx := context.Background()

	t.Run("list pending events", func(t *testing.T) {
		eventID1 := uuid.New()
		eventID2 := uuid.New()
		eventData := json.RawMessage(`{"action":"created"}`)
		createdAt1 := time.Now()
		createdAt2 := time.Now().Add(1 * time.Minute)

		rows := sqlmock.NewRows([]string{"id", "event_type", "event_data", "status", "created_at", "processed_at"}).
			AddRow(eventID1, "product.created", eventData, "pending", createdAt1, nil).
			AddRow(eventID2, "product.deleted", eventData, "pending", createdAt2, nil)

		query := repository.NewQuery()
		query.Limit = 10

		mock.ExpectPrepare("SELECT (.+) FROM events WHERE status").
			ExpectQuery().
			WithArgs("pending", 10).
			WillReturnRows(rows)

		results, err := repo.List(ctx, *query)

		require.NoError(t, err)
		assert.Len(t, results, 2)

		event1, ok := results[0].(*model.Event)
		require.True(t, ok)
		assert.Equal(t, eventID1, event1.ID)
		assert.Equal(t, "product.created", event1.EventType)

		event2, ok := results[1].(*model.Event)
		require.True(t, ok)
		assert.Equal(t, eventID2, event2.ID)
		assert.Equal(t, "product.deleted", event2.EventType)

		assert.NoError(t, mock.ExpectationsWereMet())
	})
}

func TestEventRepository_UpdateStatus(t *testing.T) {
	db, mock, err := sqlmock.New()
	require.NoError(t, err)
	defer db.Close()

	eventRepo := sql.NewEventRepository(db).(*sql.EventRepository)
	ctx := context.Background()

	t.Run("successful status update", func(t *testing.T) {
		eventID := uuid.New()

		mock.ExpectPrepare("UPDATE events SET status").
			ExpectExec().
			WithArgs("processed", eventID).
			WillReturnResult(sqlmock.NewResult(0, 1))

		err := eventRepo.UpdateStatus(ctx, eventID, model.EventStatusProcessed)

		require.NoError(t, err)
		assert.NoError(t, mock.ExpectationsWereMet())
	})
}
