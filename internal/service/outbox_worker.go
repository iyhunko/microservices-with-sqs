package service

import (
	"context"
	"encoding/json"
	"log/slog"
	"time"

	"github.com/iyhunko/microservices-with-sqs/internal/model"
	"github.com/iyhunko/microservices-with-sqs/internal/repository"
	"github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

// OutboxWorker polls the events table and processes pending events
type OutboxWorker struct {
	eventRepo      repository.Repository
	eventUpdater   repository.EventStatusUpdater
	publisher      *sqs.Publisher
	interval       time.Duration
	stopChan       chan struct{}
}

// NewOutboxWorker creates a new OutboxWorker
func NewOutboxWorker(eventRepo repository.Repository, eventUpdater repository.EventStatusUpdater, publisher *sqs.Publisher, interval time.Duration) *OutboxWorker {
	return &OutboxWorker{
		eventRepo:    eventRepo,
		eventUpdater: eventUpdater,
		publisher:    publisher,
		interval:     interval,
		stopChan:     make(chan struct{}),
	}
}

// Start begins processing events from the outbox
func (w *OutboxWorker) Start(ctx context.Context) {
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	slog.Info("Outbox worker started", slog.Duration("interval", w.interval))

	for {
		select {
		case <-ctx.Done():
			slog.Info("Outbox worker stopped by context")
			return
		case <-w.stopChan:
			slog.Info("Outbox worker stopped")
			return
		case <-ticker.C:
			w.processEvents(ctx)
		}
	}
}

// Stop stops the outbox worker
func (w *OutboxWorker) Stop() {
	close(w.stopChan)
}

// processEvents retrieves and processes pending events
func (w *OutboxWorker) processEvents(ctx context.Context) {
	// Retrieve pending events
	query := repository.NewQuery()
	query.Limit = 100
	resources, err := w.eventRepo.List(ctx, *query)
	if err != nil {
		slog.Error("Failed to retrieve pending events", slog.Any("err", err))
		return
	}

	if len(resources) == 0 {
		return
	}

	slog.Info("Processing pending events", slog.Int("count", len(resources)))

	// Process each event
	for _, resource := range resources {
		event, ok := resource.(*model.Event)
		if !ok {
			slog.Error("Invalid event type in outbox")
			continue
		}

		if err := w.processEvent(ctx, event); err != nil {
			slog.Error("Failed to process event",
				slog.String("event_id", event.ID.String()),
				slog.String("event_type", event.EventType),
				slog.Any("err", err))

			// Mark event as failed
			if updateErr := w.eventUpdater.UpdateStatus(ctx, event.ID, model.EventStatusFailed); updateErr != nil {
				slog.Error("Failed to update event status to failed",
					slog.String("event_id", event.ID.String()),
					slog.Any("err", updateErr))
			}
		} else {
			// Mark event as processed
			if updateErr := w.eventUpdater.UpdateStatus(ctx, event.ID, model.EventStatusProcessed); updateErr != nil {
				slog.Error("Failed to update event status to processed",
					slog.String("event_id", event.ID.String()),
					slog.Any("err", updateErr))
			} else {
				slog.Info("Event processed successfully",
					slog.String("event_id", event.ID.String()),
					slog.String("event_type", event.EventType))
			}
		}
	}
}

// processEvent publishes a single event to SQS
func (w *OutboxWorker) processEvent(ctx context.Context, event *model.Event) error {
	// Parse event data into ProductMessage
	var productMsg sqs.ProductMessage
	if err := json.Unmarshal(event.EventData, &productMsg); err != nil {
		return err
	}

	// Publish message to SQS
	if err := w.publisher.PublishProductMessage(ctx, productMsg); err != nil {
		return err
	}

	return nil
}
