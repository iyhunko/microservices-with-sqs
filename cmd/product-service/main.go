package product_service

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/aws/aws-sdk-go-v2/aws"
	awsconfig "github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/sqs"
	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	"github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"github.com/iyhunko/microservices-with-sqs/internal/service"
	sqspkg "github.com/iyhunko/microservices-with-sqs/internal/sqs"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	conf, err := config.LoadFromEnv()
	handleErr("loading config", err)

	ctx := context.Background()
	db, err := sql.StartDB(ctx, conf.Database)
	handleErr("starting database", err)

	// Create repositories
	userRepository := sql.NewUserRepository(db)
	productRepository := sql.NewProductRepository(db)
	eventRepository := sql.NewEventRepository(db)
	transactionalRepository := sql.NewTransactionalRepository(db)

	// Initialize AWS SQS client (required for product service)
	// SQSQueueURL is now a required configuration parameter
	awsCfg, err := awsconfig.LoadDefaultConfig(ctx,
		awsconfig.WithRegion(conf.AWS.Region),
	)
	handleErr("loading AWS config", err)

	// Override endpoint for LocalStack if specified
	if conf.AWS.Endpoint != "" {
		awsCfg.BaseEndpoint = aws.String(conf.AWS.Endpoint)
	}

	sqsClient := sqs.NewFromConfig(awsCfg)
	sqsPublisher := sqspkg.NewPublisher(sqsClient, conf.AWS.SQSQueueURL)

	// Create services with outbox pattern
	productService := service.NewProductServiceWithOutbox(productRepository, transactionalRepository, sqsPublisher)

	// Start outbox worker to process pending events every 2 seconds
	outboxWorker := service.NewOutboxWorker(eventRepository.(*sql.EventRepository), sqsPublisher, 2*time.Second)
	go outboxWorker.Start(ctx)

	// Start HTTP server
	ctr := controller.New(conf, userRepository)
	productCtr := controller.NewProductController(productService)
	httpServer := gin.Default()
	httpServer = httpAPI.InitRouter(conf, userRepository, httpServer, ctr, productCtr)

	go func() {
		err = httpServer.Run(":" + conf.HTTPServer.Port)
		if err != nil {
			handleErr("listening to HTTP requests", err)
		}
	}()

	// Start metrics server
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics server starting on port %s", conf.MetricsServer.Port)
		if err := http.ListenAndServe(":"+conf.MetricsServer.Port, nil); err != nil {
			handleErr("listening to metrics requests", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	log.Println("Shutting down gracefully...")
	// TODO: stop httpServer gracefully
}

func handleErr(msg string, err error) {
	if err != nil {
		log.Fatalf("error while %s: %v", msg, err)
	}
}
