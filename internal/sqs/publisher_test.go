package sqs

import (
	"context"
	"errors"
	"testing"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// mockSQSClient is a mock implementation of the SQS client for testing.
type mockSQSClient struct {
	sendMessageFunc func(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error)
}

func (m *mockSQSClient) SendMessage(ctx context.Context, params *sqs.SendMessageInput, optFns ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
	if m.sendMessageFunc != nil {
		return m.sendMessageFunc(ctx, params, optFns...)
	}
	return &sqs.SendMessageOutput{}, nil
}

func TestPublisher_PublishProductMessage(t *testing.T) {
	t.Run("successful message publish", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		mockClient := &mockSQSClient{
			sendMessageFunc: func(_ context.Context, params *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				assert.Equal(t, queueURL, *params.QueueUrl)
				assert.NotNil(t, params.MessageBody)
				return &sqs.SendMessageOutput{
					MessageId: aws.String("test-message-id"),
				}, nil
			},
		}

		publisher := &Publisher{
			client:   mockClient,
			queueURL: queueURL,
		}

		msg := ProductMessage{
			Action:    "created",
			ProductID: "123",
			Name:      "Test Product",
			Price:     99.99,
		}

		// when
		err := publisher.PublishProductMessage(ctx, msg)

		// then
		require.NoError(t, err)
	})

	t.Run("error sending message", func(t *testing.T) {
		// given
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		ctx := context.Background()

		expectedErr := errors.New("failed to send message")
		mockClient := &mockSQSClient{
			sendMessageFunc: func(_ context.Context, _ *sqs.SendMessageInput, _ ...func(*sqs.Options)) (*sqs.SendMessageOutput, error) {
				return nil, expectedErr
			},
		}

		publisher := &Publisher{
			client:   mockClient,
			queueURL: queueURL,
		}

		msg := ProductMessage{
			Action:    "created",
			ProductID: "123",
			Name:      "Test Product",
			Price:     99.99,
		}

		// when
		err := publisher.PublishProductMessage(ctx, msg)

		// then
		require.Error(t, err)
		assert.Contains(t, err.Error(), "failed to send message to SQS")
	})
}

func TestNewPublisher(t *testing.T) {
	t.Run("creates publisher successfully", func(t *testing.T) {
		// given
		mockClient := &mockSQSClient{}
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"

		// when
		publisher := NewPublisher(mockClient, queueURL)

		// then
		require.NotNil(t, publisher)
		assert.Equal(t, queueURL, publisher.queueURL)
	})
}
