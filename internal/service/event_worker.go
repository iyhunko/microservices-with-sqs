package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	reposql "github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

// EventWorker handles processing of pending events from the outbox table.
type EventWorker struct {
	eventRepo *reposql.EventRepository
	publisher *sqs.Publisher
	interval  time.Duration
}

// NewEventWorker creates a new EventWorker instance.
func NewEventWorker(eventRepo *reposql.EventRepository, publisher *sqs.Publisher, interval time.Duration) *EventWorker {
	return &EventWorker{
		eventRepo: eventRepo,
		publisher: publisher,
		interval:  interval,
	}
}

// Start begins the worker loop that processes pending events.
func (ew *EventWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(ew.interval)
	defer ticker.Stop()

	slog.Info("Event worker started", slog.Duration("interval", ew.interval))

	for {
		select {
		case <-ctx.Done():
			slog.Info("Event worker stopping")
			return
		case <-ticker.C:
			if err := ew.processPendingEvents(ctx); err != nil {
				slog.Error("Failed to process pending events", slog.Any("err", err))
			}
		}
	}
}

// processPendingEvents fetches and processes all pending events.
func (ew *EventWorker) processPendingEvents(ctx context.Context) error {
	// Query for pending events
	query := repository.NewQuery().With(repository.StatusField, string(model.EventStatusPending))
	query.Limit = 100 // Process up to 100 events at a time

	resources, err := ew.eventRepo.List(ctx, *query)
	if err != nil {
		return err
	}

	for _, resource := range resources {
		event, ok := resource.(*model.Event)
		if !ok {
			slog.Error("Invalid resource type", slog.Any("resource", resource))
			continue
		}

		if err := ew.processEvent(ctx, event); err != nil {
			slog.Error("Failed to process event", slog.String("event_id", event.ID.String()), slog.Any("err", err))
			// Mark event as failed
			if updateErr := ew.eventRepo.UpdateStatus(ctx, event.ID, model.EventStatusFailed); updateErr != nil {
				slog.Error("Failed to update event status to failed", slog.String("event_id", event.ID.String()), slog.Any("err", updateErr))
			}
		} else {
			// Mark event as processed
			if updateErr := ew.eventRepo.UpdateStatus(ctx, event.ID, model.EventStatusProcessed); updateErr != nil {
				slog.Error("Failed to update event status to processed", slog.String("event_id", event.ID.String()), slog.Any("err", updateErr))
			}
		}
	}

	return nil
}

// processEvent processes a single event by publishing it to SQS.
func (ew *EventWorker) processEvent(ctx context.Context, event *model.Event) error {
	// Parse event data
	var msg sqs.ProductMessage
	if err := json.Unmarshal(event.EventData, &msg); err != nil {
		return err
	}

	// Publish to SQS
	if ew.publisher != nil {
		if err := ew.publisher.PublishProductMessage(ctx, msg); err != nil {
			return err
		}
		slog.Info("Event published to SQS", slog.String("event_id", event.ID.String()), slog.String("event_type", event.EventType))
	}

	return nil
}


