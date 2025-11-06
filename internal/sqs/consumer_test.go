package sqs

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSQSConsumerClient is a mock implementation of the SQS client for consumer testing.
type mockSQSConsumerClient struct {
	receiveMessageFunc func(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error)
	deleteMessageFunc  func(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error)
}

func (m *mockSQSConsumerClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, optFns ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	if m.receiveMessageFunc != nil {
		return m.receiveMessageFunc(ctx, params, optFns...)
	}
	return &sqs.ReceiveMessageOutput{Messages: []types.Message{}}, nil
}

func (m *mockSQSConsumerClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, optFns ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	if m.deleteMessageFunc != nil {
		return m.deleteMessageFunc(ctx, params, optFns...)
	}
	return &sqs.DeleteMessageOutput{}, nil
}

func TestConsumer_processMessage(t *testing.T) {
	t.Run("successful message processing", func(t *testing.T) {
		// given
		consumer := &Consumer{
			queueURL: "https://sqs.us-east-1.amazonaws.com/123456789/test-queue",
		}

		messageBody := `{"action":"created","product_id":"123","name":"Test Product","price":99.99}`
		message := types.Message{
			Body:          aws.String(messageBody),
			ReceiptHandle: aws.String("test-receipt-handle"),
		}

		// when
		err := consumer.processMessage(context.Background(), message)

		// then
		require.NoError(t, err)
	})

	t.Run("nil message body", func(t *testing.T) {
		// given
		consumer := &Consumer{
			queueURL: "https://sqs.us-east-1.amazonaws.com/123456789/test-queue",
		}

		message := types.Message{
			Body:          nil,
			ReceiptHandle: aws.String("test-receipt-handle"),
		}

		// when
		err := consumer.processMessage(context.Background(), message)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "message body is nil")
	})

	t.Run("invalid JSON message body", func(t *testing.T) {
		// given
		consumer := &Consumer{
			queueURL: "https://sqs.us-east-1.amazonaws.com/123456789/test-queue",
		}

		messageBody := `{"invalid json`
		message := types.Message{
			Body:          aws.String(messageBody),
			ReceiptHandle: aws.String("test-receipt-handle"),
		}

		// when
		err := consumer.processMessage(context.Background(), message)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal message")
	})
}

func TestConsumer_deleteMessage(t *testing.T) {
	t.Run("successful message deletion", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		mockClient := &mockSQSConsumerClient{
			deleteMessageFunc: func(_ context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				assert.Equal(t, queueURL, *params.QueueUrl)
				assert.NotNil(t, params.ReceiptHandle)
				return &sqs.DeleteMessageOutput{}, nil
			},
		}

		consumer := &Consumer{
			client:   mockClient,
			queueURL: queueURL,
		}

		message := types.Message{
			ReceiptHandle: aws.String("test-receipt-handle"),
		}

		// when
		err := consumer.deleteMessage(ctx, message)

		// then
		require.NoError(t, err)
	})

	t.Run("error deleting message", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		expectedErr := errors.New("failed to delete")
		mockClient := &mockSQSConsumerClient{
			deleteMessageFunc: func(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return nil, expectedErr
			},
		}

		consumer := &Consumer{
			client:   mockClient,
			queueURL: queueURL,
		}

		message := types.Message{
			ReceiptHandle: aws.String("test-receipt-handle"),
		}

		// when
		err := consumer.deleteMessage(ctx, message)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to delete message")
	})
}

func TestConsumer_receiveMessages(t *testing.T) {
	t.Run("receives and processes messages successfully", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		messageBody := `{"action":"created","product_id":"123","name":"Test Product","price":99.99}`
		mockClient := &mockSQSConsumerClient{
			receiveMessageFunc: func(_ context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				assert.Equal(t, queueURL, *params.QueueUrl)
				assert.Equal(t, int32(10), params.MaxNumberOfMessages)
				assert.Equal(t, int32(20), params.WaitTimeSeconds)
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String(messageBody),
							ReceiptHandle: aws.String("test-receipt-handle"),
						},
					},
				}, nil
			},
			deleteMessageFunc: func(_ context.Context, _ *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
				return &sqs.DeleteMessageOutput{}, nil
			},
		}

		consumer := &Consumer{
			client:   mockClient,
			queueURL: queueURL,
		}

		// when
		err := consumer.receiveMessages(ctx)

		// then
		require.NoError(t, err)
	})

	t.Run("handles receive message error", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		expectedErr := errors.New("failed to receive")
		mockClient := &mockSQSConsumerClient{
			receiveMessageFunc: func(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return nil, expectedErr
			},
		}

		consumer := &Consumer{
			client:   mockClient,
			queueURL: queueURL,
		}

		// when
		err := consumer.receiveMessages(ctx)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to receive messages")
	})

	t.Run("continues processing on message processing error", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		// Invalid JSON to trigger processing error
		invalidMessageBody := `{"invalid json`
		mockClient := &mockSQSConsumerClient{
			receiveMessageFunc: func(_ context.Context, _ *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
				return &sqs.ReceiveMessageOutput{
					Messages: []types.Message{
						{
							Body:          aws.String(invalidMessageBody),
							ReceiptHandle: aws.String("test-receipt-handle"),
						},
					},
				}, nil
			},
		}

		consumer := &Consumer{
			client:   mockClient,
			queueURL: queueURL,
		}

		// when
		err := consumer.receiveMessages(ctx)

		// then
		// Should not return error - processing errors are logged but don't stop the consumer
		require.NoError(t, err)
	})
}

func TestNewConsumer(t *testing.T) {
	t.Run("creates consumer successfully", func(t *testing.T) {
		// given
		mockClient := &mockSQSConsumerClient{}
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"

		// when
		consumer := NewConsumer(mockClient, queueURL)

		// then
		require.NotNil(t, consumer)
		assert.Equal(t, queueURL, consumer.queueURL)
	})
}
