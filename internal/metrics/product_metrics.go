package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	// ProductsCreated is a Prometheus counter for tracking the total number of products created.
	ProductsCreated = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_created_total",
		Help: "The total number of products created",
	})

	// ProductsDeleted is a Prometheus counter for tracking the total number of products deleted.
	ProductsDeleted = promauto.NewCounter(prometheus.CounterOpts{
		Name: "products_deleted_total",
		Help: "The total number of products deleted",
	})
)
