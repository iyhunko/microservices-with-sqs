package main

import (
	"context"
	"errors"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/iyhunko/microservices-with-sqs/internal/logger"
	sqspkg "github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

func main() {
	// Initialize JSON logger for structured logging
	logger.InitJSONLogger()

	conf, err := config.LoadFromEnv()
	handleErr("loading config", err)

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// Initialize AWS SQS client
	sqsClient, err := sqspkg.NewClient(ctx, conf.AWS.Region, conf.AWS.Endpoint)
	handleErr("creating SQS client", err)

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
