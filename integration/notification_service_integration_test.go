package integration

import (
	"context"
	"encoding/json"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/aws/aws-sdk-go-v2/service/sqs/types"
	sqspkg "github.com/iyhunko/microservices-with-sqs/internal/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

// MockSQSClient implements the ConsumerAPI interface for testing.
type MockSQSClient struct {
	mock.Mock
}

func (m *MockSQSClient) ReceiveMessage(ctx context.Context, params *sqs.ReceiveMessageInput, _ ...func(*sqs.Options)) (*sqs.ReceiveMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.ReceiveMessageOutput), args.Error(1)
}

func (m *MockSQSClient) DeleteMessage(ctx context.Context, params *sqs.DeleteMessageInput, _ ...func(*sqs.Options)) (*sqs.DeleteMessageOutput, error) {
	args := m.Called(ctx, params)
	if args.Get(0) == nil {
		return nil, args.Error(1)
	}
	return args.Get(0).(*sqs.DeleteMessageOutput), args.Error(1)
}

func TestNotificationService_Integration(t *testing.T) {
	t.Run("consumer receives and processes product created message", func(t *testing.T) {
		mockClient := new(MockSQSClient)
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		consumer := sqspkg.NewConsumer(mockClient, queueURL)

		// Create a test product message
		productMsg := sqspkg.ProductMessage{
			Action:    "created",
			ProductID: "123e4567-e89b-12d3-a456-426614174000",
			Name:      "Test Product",
			Price:     99.99,
		}
		msgBody, err := json.Marshal(productMsg)
		require.NoError(t, err)

		receiptHandle := "test-receipt-handle"
		messageBody := string(msgBody)

		// Setup mock expectations
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{
				Messages: []types.Message{
					{
						Body:          &messageBody,
						ReceiptHandle: &receiptHandle,
					},
				},
			},
			nil,
		).Once()

		mockClient.On("DeleteMessage", mock.Anything, mock.MatchedBy(func(params *sqs.DeleteMessageInput) bool {
			return *params.ReceiptHandle == receiptHandle
		})).Return(&sqs.DeleteMessageOutput{}, nil).Once()

		// Return empty messages on second call to avoid infinite loop
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: []types.Message{}},
			nil,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		// Start consuming in a goroutine
		done := make(chan error, 1)
		go func() {
			done <- consumer.Start(ctx)
		}()

		// Wait for context to timeout or completion
		select {
		case err := <-done:
			assert.Error(t, err) // Should return context.DeadlineExceeded
		case <-time.After(3 * time.Second):
			t.Fatal("Test timed out")
		}

		// Verify that the message was received and deleted
		mockClient.AssertExpectations(t)
	})

	t.Run("consumer receives and processes product deleted message", func(t *testing.T) {
		mockClient := new(MockSQSClient)
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		consumer := sqspkg.NewConsumer(mockClient, queueURL)

		// Create a test product deleted message
		productMsg := sqspkg.ProductMessage{
			Action:    "deleted",
			ProductID: "123e4567-e89b-12d3-a456-426614174000",
			Name:      "Deleted Product",
			Price:     49.99,
		}
		msgBody, err := json.Marshal(productMsg)
		require.NoError(t, err)

		receiptHandle := "test-receipt-handle-2"
		messageBody := string(msgBody)

		// Setup mock expectations
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{
				Messages: []types.Message{
					{
						Body:          &messageBody,
						ReceiptHandle: &receiptHandle,
					},
				},
			},
			nil,
		).Once()

		mockClient.On("DeleteMessage", mock.Anything, mock.MatchedBy(func(params *sqs.DeleteMessageInput) bool {
			return *params.ReceiptHandle == receiptHandle
		})).Return(&sqs.DeleteMessageOutput{}, nil).Once()

		// Return empty messages on second call
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: []types.Message{}},
			nil,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- consumer.Start(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("Test timed out")
		}

		mockClient.AssertExpectations(t)
	})

	t.Run("consumer handles invalid message gracefully", func(t *testing.T) {
		mockClient := new(MockSQSClient)
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		consumer := sqspkg.NewConsumer(mockClient, queueURL)

		receiptHandle := "test-receipt-handle-3"
		invalidMessageBody := "invalid json message"

		// Setup mock expectations - invalid message should not be deleted
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{
				Messages: []types.Message{
					{
						Body:          &invalidMessageBody,
						ReceiptHandle: &receiptHandle,
					},
				},
			},
			nil,
		).Once()

		// The invalid message should NOT result in a DeleteMessage call
		// because processing failed

		// Return empty messages on subsequent calls
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: []types.Message{}},
			nil,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- consumer.Start(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("Test timed out")
		}

		mockClient.AssertExpectations(t)
	})

	t.Run("consumer processes multiple messages in batch", func(t *testing.T) {
		mockClient := new(MockSQSClient)
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		consumer := sqspkg.NewConsumer(mockClient, queueURL)

		// Create multiple test messages
		messages := []types.Message{}
		for i := 0; i < 3; i++ {
			productMsg := sqspkg.ProductMessage{
				Action:    "created",
				ProductID: "123e4567-e89b-12d3-a456-42661417400" + string(rune('0'+i)),
				Name:      "Product " + string(rune('A'+i)),
				Price:     float64(10 * (i + 1)),
			}
			msgBody, _ := json.Marshal(productMsg)
			messageBody := string(msgBody)
			receiptHandle := "receipt-" + string(rune('0'+i))
			messages = append(messages, types.Message{
				Body:          &messageBody,
				ReceiptHandle: &receiptHandle,
			})
		}

		// Setup mock expectations for receiving all messages
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: messages},
			nil,
		).Once()

		// Expect DeleteMessage to be called for each message
		for _, msg := range messages {
			mockClient.On("DeleteMessage", mock.Anything, mock.MatchedBy(func(params *sqs.DeleteMessageInput) bool {
				return *params.ReceiptHandle == *msg.ReceiptHandle
			})).Return(&sqs.DeleteMessageOutput{}, nil).Once()
		}

		// Return empty messages on subsequent calls
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: []types.Message{}},
			nil,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- consumer.Start(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("Test timed out")
		}

		mockClient.AssertExpectations(t)
	})

	t.Run("consumer handles nil message body", func(t *testing.T) {
		mockClient := new(MockSQSClient)
		queueURL := "https://sqs.us-east-1.amazonaws.com/123456789/test-queue"
		consumer := sqspkg.NewConsumer(mockClient, queueURL)

		receiptHandle := "test-receipt-handle-4"

		// Setup mock expectations with nil body
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{
				Messages: []types.Message{
					{
						Body:          nil,
						ReceiptHandle: &receiptHandle,
					},
				},
			},
			nil,
		).Once()

		// Nil body should not result in deletion

		// Return empty messages on subsequent calls
		mockClient.On("ReceiveMessage", mock.Anything, mock.Anything).Return(
			&sqs.ReceiveMessageOutput{Messages: []types.Message{}},
			nil,
		)

		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()

		done := make(chan error, 1)
		go func() {
			done <- consumer.Start(ctx)
		}()

		select {
		case err := <-done:
			assert.Error(t, err)
		case <-time.After(3 * time.Second):
			t.Fatal("Test timed out")
		}

		mockClient.AssertExpectations(t)
	})
}
