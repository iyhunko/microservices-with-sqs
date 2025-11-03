package product_service

import (
	"context"
	"github.com/gin-gonic/gin"
	"github.com/iyhunko/microservices-with-sqs/internal/config"
	httpAPI "github.com/iyhunko/microservices-with-sqs/internal/http"
	"github.com/iyhunko/microservices-with-sqs/internal/http/controller"
	"github.com/iyhunko/microservices-with-sqs/internal/repository/sql"
	"log"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	conf, err := config.LoadFromEnv()
	handleErr("loading config", err)

	ctx := context.Background()
	db, err := sql.StartDB(ctx, conf.Database)
	handleErr("starting database", err)

	repository := sql.NewRepository(db)

	// start http server
	ctr := controller.New(conf, repository)
	httpServer := gin.Default()
	httpServer = httpAPI.InitRouter(conf, repository, httpServer, ctr)

	go func() {
		err = httpServer.Run(":" + conf.HTTPServer.Port)
		if err != nil {
			handleErr("listening to Http requests", err)
		}
	}()

	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, syscall.SIGINT, syscall.SIGTERM)
	<-sigChan
	// TODO: stop httpServer gracefully
}

func handleErr(msg string, err error) {
	if err != nil {
		log.Fatalf("error while %s: %v", msg, err)
	}
}
