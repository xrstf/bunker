package main

import (
	"fmt"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	requestsProcessed = promauto.NewCounterVec(prometheus.CounterOpts{
		Name: "bunker_requests_total",
		Help: "The total number of handled requests",
	}, []string{"status"})

	recordsReceived = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bunker_received_records_total",
		Help: "The total number of received records",
	})

	recordsIngested = promauto.NewCounter(prometheus.CounterOpts{
		Name: "bunker_ingested_records_total",
		Help: "The total number of ingested records",
	})
)

func metricsMiddleware(handler echo.HandlerFunc) echo.HandlerFunc {
	return func(c echo.Context) error {
		err := handler(c)

		statusCode := fmt.Sprintf("%d", c.Response().Status)
		requestsProcessed.WithLabelValues(statusCode).Inc()

		return err
	}
}
