package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	"github.com/iyhunko/microservices-with-sqs/internal/logger"
	"github.com/iyhunko/microservices-with-sqs/internal/metrics"
	"github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	sqspkg "github.com/iyhunko/microservices-with-sqs/internal/sqs"
)

func main() {
	// Initialize JSON logger for structured logging
	logger.InitJSONLogger()

	conf, err := config.LoadFromEnv()
	handleErr("loading config", err)

	ctx := context.Background()
	db, err := sql.StartDB(ctx, conf.Database)
	handleErr("starting database", err)

	// Create repositories
	userRepository := sql.NewUserRepository(db)
	productRepository := sql.NewProductRepository(db)
	eventRepository := sql.NewEventRepository(db)

	// Initialize AWS SQS client (required for product service)
	sqsClient, err := sqspkg.NewClient(ctx, conf.AWS.Region, conf.AWS.Endpoint)
	handleErr("creating SQS client", err)

	sqsPublisher := sqspkg.NewPublisher(sqsClient, conf.AWS.SQSQueueURL)

	// Create services
	productService := service.NewProductService(db, productRepository, eventRepository, sqsPublisher)

	// Start HTTP server
	productCtr := controller.NewProductController(productService)
	httpServer := gin.Default()
	httpServer = httpAPI.InitRouter(conf, userRepository, httpServer, productCtr)

	go func() {
		err = httpServer.Run(":" + conf.HTTPServer.Port)
		if err != nil {
			handleErr("listening to HTTP requests", err)
		}
	}()

	// Start metrics server
	metrics.StartMetricsServer(conf)

	// Start event worker (outbox pattern)
	eventWorker := service.NewEventWorker(eventRepository, sqsPublisher, 2*time.Second)
	workerCtx, workerCancel := context.WithCancel(ctx)
	defer workerCancel()
	go eventWorker.Start(workerCtx)

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
	workerCancel() // Stop the event worker
}

func handleErr(msg string, err error) {
	if err != nil {
		log.Fatalf("error while %s: %v", msg, err)
	}
}
