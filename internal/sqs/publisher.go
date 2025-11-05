package sqs

import (
	"context"
	"encoding/json"
	"fmt"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
)

type Publisher struct {
	client   *sqs.Client
	queueURL string
}

func NewPublisher(client *sqs.Client, queueURL string) *Publisher {
	return &Publisher{
		client:   client,
		queueURL: queueURL,
	}
}

type ProductMessage struct {
	Action    string  `json:"action"`
	ProductID string  `json:"product_id"`
	Name      string  `json:"name"`
	Price     float64 `json:"price"`
}

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
