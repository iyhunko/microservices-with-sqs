package sqs

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
)

// ConsumerAPI defines the interface for SQS operations used by Consumer.
type ConsumerAPI interface {
	ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

// Consumer handles consuming messages from AWS SQS.
type Consumer struct {
	client   ConsumerAPI
	queueURL string
}

// NewConsumer creates a new SQS Consumer with the given client and queue URL.
func NewConsumer(client ConsumerAPI, queueURL string) *Consumer {
	return &Consumer{
		client:   client,
		queueURL: queueURL,
	}
}

// Start begins consuming messages from the SQS queue until the context is cancelled.
func (c *Consumer) Start(ctx context.Context) error {
	slog.Info("Starting SQS consumer", slog.String("queueURL", c.queueURL))

	for {
		select {
		case <-ctx.Done():
			slog.Info("Stopping SQS consumer")
			return ctx.Err()
		default:
			if err := c.receiveMessages(ctx); err != nil {
				slog.Error("Error receiving messages", slog.Any("err", err))
			}
		}
	}
}

func (c *Consumer) receiveMessages(ctx context.Context) error {
	result, err := c.client.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
		QueueUrl:            aws.String(c.queueURL),
		MaxNumberOfMessages: 10,
		WaitTimeSeconds:     20, // Long polling
	})
	if err != nil {
		return fmt.Errorf("failed to receive messages: %w", err)
	}

	for _, message := range result.Messages {
		if err := c.processMessage(ctx, message); err != nil {
			slog.Error("Error processing message", slog.Any("err", err))
			continue
		}

		// Delete message after successful processing
		if err := c.deleteMessage(ctx, message); err != nil {
			slog.Error("Error deleting message", slog.Any("err", err))
		}
	}

	return nil
}

func (c *Consumer) processMessage(_ context.Context, message types.Message) error {
	if message.Body == nil {
		return fmt.Errorf("message body is nil")
	}

	var productMsg ProductMessage
	if err := json.Unmarshal([]byte(*message.Body), &productMsg); err != nil {
		return fmt.Errorf("failed to unmarshal message: %w", err)
	}

	// Log the received message
	slog.Info("Received product notification",
		slog.String("action", productMsg.Action),
		slog.String("product_id", productMsg.ProductID),
		slog.String("name", productMsg.Name),
		slog.Float64("price", productMsg.Price),
	)

	return nil
}

func (c *Consumer) deleteMessage(ctx context.Context, message types.Message) error {
	_, err := c.client.DeleteMessage(ctx, &sqs.DeleteMessageInput{
		QueueUrl:      aws.String(c.queueURL),
		ReceiptHandle: message.ReceiptHandle,
	})
	if err != nil {
		return fmt.Errorf("failed to delete message: %w", err)
	}
	return nil
}
