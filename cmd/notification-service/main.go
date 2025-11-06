package notification_service

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	sqspkg "github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

func main() {
	conf, err := config.LoadFromEnv()
	handleErr("loading config", err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize AWS SQS client
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(conf.AWS.Region),
	)
	handleErr("loading AWS config", err)

	// Override endpoint for LocalStack if specified
	if conf.AWS.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(conf.AWS.Endpoint)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)
	consumer := sqspkg.NewConsumer(sqsClient, conf.AWS.SQSQueueURL)

	// Start consuming messages
	go func() {
		if err := consumer.Start(ctx); err != nil && !errors.Is(err, context.Canceled) {
			log.Printf("Consumer error: %v", err)
		}
	}()

	log.Println("Notification service started. Listening for messages...")

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
	cancel()
}

func handleErr(msg string, err error) {
	if err != nil {
		log.Fatalf("error while %s: %v", msg, err)
	}
}
