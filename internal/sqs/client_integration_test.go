package sqs

import (
	"context"
	"encoding/json"
	"os"
	"testing"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestClient_Integration_WithLocalStack tests the SQS client with LocalStack endpoint.
// This test requires LocalStack to be running on localhost:4566 with the product-notifications queue created.
// Run with: go test -v -run Integration ./internal/sqs/...
func TestClient_Integration_WithLocalStack(t *testing.T) {
	// Skip if not running integration tests
	if testing.Short() {
		t.Skip("Skipping integration test")
	}

	// Check if LocalStack is available by checking environment or skipping
	endpoint := os.Getenv("AWS_ENDPOINT")
	if endpoint == "" {
		endpoint = "http://localhost:4566"
	}

	queueURL := os.Getenv("SQS_QUEUE_URL")
	if queueURL == "" {
		queueURL = "http://localhost:4566/000000000000/product-notifications"
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	t.Run("create SQS client with custom endpoint", func(t *testing.T) {
		// Create SQS client with custom endpoint (LocalStack)
		sqsClient, err := NewClient(ctx, "us-east-1", endpoint)
		require.NoError(t, err)
		require.NotNil(t, sqsClient)

		// Try to list queues to verify connection
		listOutput, err := sqsClient.ListQueues(ctx, &sqs.ListQueuesInput{})
		if err != nil {
			// If LocalStack is not running, skip the test
			t.Skipf("LocalStack not available: %v", err)
		}

		// Verify we can see the product-notifications queue
		require.NotNil(t, listOutput.QueueUrls)
		t.Logf("Found %d queue(s)", len(listOutput.QueueUrls))
	})

	t.Run("publish and receive message with custom endpoint", func(t *testing.T) {
		// Create SQS client with custom endpoint
		sqsClient, err := NewClient(ctx, "us-east-1", endpoint)
		require.NoError(t, err)

		// Create publisher
		publisher := NewPublisher(sqsClient, queueURL)

		// Create a test message
		msg := ProductMessage{
			Action:    "created",
			ProductID: "test-product-integration-123",
			Name:      "Integration Test Product",
			Price:     99.99,
		}

		// Publish the message
		err = publisher.PublishProductMessage(ctx, msg)
		if err != nil {
			// If LocalStack is not running or queue doesn't exist, skip the test
			t.Skipf("Failed to publish message (LocalStack may not be running): %v", err)
		}

		// Wait a bit for the message to be available
		time.Sleep(100 * time.Millisecond)

		// Try to receive the message
		output, err := sqsClient.ReceiveMessage(ctx, &sqs.ReceiveMessageInput{
			QueueUrl:            aws.String(queueURL),
			MaxNumberOfMessages: 10,
			WaitTimeSeconds:     2,
		})
		require.NoError(t, err)

		// We should receive at least one message
		require.NotEmpty(t, output.Messages, "Expected at least one message in the queue")

		// Find our message
		var found bool
		for _, sqsMsg := range output.Messages {
			var receivedMsg ProductMessage
			err = json.Unmarshal([]byte(*sqsMsg.Body), &receivedMsg)
			if err == nil && receivedMsg.ProductID == msg.ProductID {
				found = true
				assert.Equal(t, msg.Action, receivedMsg.Action)
				assert.Equal(t, msg.Name, receivedMsg.Name)
				assert.Equal(t, msg.Price, receivedMsg.Price)

				// Clean up: delete the message
				_, err := sqsClient.DeleteMessage(ctx, &sqs.DeleteMessageInput{
					QueueUrl:      aws.String(queueURL),
					ReceiptHandle: sqsMsg.ReceiptHandle,
				})
				assert.NoError(t, err)
				break
			}
		}

		assert.True(t, found, "Did not find our test message in the queue")
	})
}
