package metrics

import (
	"github.com/gofiber/fiber/v2"
	"github.com/gofiber/fiber/v2/middleware/adaptor"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"log"
)

var (
	httpRequestsTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: "http_requests_total",
			Help: "Total number of HTTP requests",
		},
		[]string{"method", "path", "status"},
	)

	httpRequestDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "http_request_duration_seconds",
			Help:    "Duration of HTTP requests in seconds",
			Buckets: []float64{0.1, 0.3, 1.2, 5, 10},
		},
		[]string{"method", "path"},
	)
)

func init() {
	prometheus.MustRegister(httpRequestsTotal)
	prometheus.MustRegister(httpRequestDuration)
	log.Println("Metrics collectors registered")
}

// MetricsMiddleware returns a Fiber middleware for metrics
func MetricsMiddleware() fiber.Handler {
	return func(c *fiber.Ctx) error {
		timer := prometheus.NewTimer(httpRequestDuration.WithLabelValues(c.Method(), c.Path()))
		defer timer.ObserveDuration()

		err := c.Next()

		status := c.Response().StatusCode()
		httpRequestsTotal.WithLabelValues(c.Method(), c.Path(), string(rune(status))).Inc()

		return err
	}
}

// MetricsHandler returns a handler for the metrics endpoint
func MetricsHandler() fiber.Handler {
	return adaptor.HTTPHandler(promhttp.Handler())
}
