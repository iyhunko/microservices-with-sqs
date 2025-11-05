package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

// Publisher handles publishing messages to AWS SQS.
type Publisher struct {
	client   *sqs.Client
	queueURL string
}

// NewPublisher creates a new SQS Publisher with the given client and queue URL.
func NewPublisher(client *sqs.Client, queueURL string) *Publisher {
	return &Publisher{
		client:   client,
		queueURL: queueURL,
	}
}

// ProductMessage represents a message about a product event.
type ProductMessage struct {
	Action    string  `json:"action"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
}

// PublishProductMessage publishes a product message to the SQS queue.
func (p *Publisher) PublishProductMessage(ctx context.Context, msg ProductMessage) error {
	messageBody, err := json.Marshal(msg)
	if err != nil {
		return fmt.Errorf("failed to marshal message: %w", err)
	}

	_, err = p.client.SendMessage(ctx, &sqs.SendMessageInput{
		QueueUrl:    aws.String(p.queueURL),
		MessageBody: aws.String(string(messageBody)),
	})
	if err != nil {
		return fmt.Errorf("failed to send message to SQS: %w", err)
	}

	return nil
}
