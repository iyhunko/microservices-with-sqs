package metrics

import (
	"log"
	"net/http"
	"time"

	"github.com/iyhunko/microservices-with-sqs/internal/config"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

// StartMetricsServer starts the metrics HTTP server on the specified port.
// It runs in a goroutine and handles the /metrics endpoint.
func StartMetricsServer(conf *config.Config) {
	go func() {
		http.Handle("/metrics", promhttp.Handler())
		log.Printf("Metrics server starting on port %s", conf.MetricsServer.Port)
		metricsServer := &http.Server{
			Addr:              ":" + conf.MetricsServer.Port,
			ReadHeaderTimeout: 5 * time.Second,
			ReadTimeout:       10 * time.Second,
			WriteTimeout:      10 * time.Second,
			IdleTimeout:       60 * time.Second,
		}
		if err := metricsServer.ListenAndServe(); err != nil {
			log.Fatalf("error while listening to metrics requests: %v", err)
		}
	}()
}
