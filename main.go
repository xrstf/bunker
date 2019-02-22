package main

import (
	"context"
	"encoding/json"
	"flag"
	"log"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promhttp"
	"github.com/sirupsen/logrus"
)

type Config struct {
	Target  string
	Pattern string
	Listen  string
	Verbose bool
}

func main() {
	config := Config{}

	flag.StringVar(&config.Target, "target", "records", "path to where incoming records should be written to")
	flag.StringVar(&config.Pattern, "pattern", "%date%/%kubernetes_namespace_name%.json", "filename pattern to group records into files")
	flag.StringVar(&config.Listen, "listen", "0.0.0.0:9095", "address and port to listen on")
	flag.BoolVar(&config.Verbose, "verbose", false, "incrases logging verbosity")
	flag.Parse()

	logger := makeLogger(&config)

	filter, err := NewFilter(&config)
	if err != nil {
		logger.Fatalf("Failed to create filter: %v", err)
	}

	sink, err := NewSink(&config, filter, logger)
	if err != nil {
		logger.Fatalf("Failed to start log processor: %v", err)
	}

	err = prometheus.Register(sink)
	if err != nil {
		logger.Fatalf("Failed to register sink metrics collector: %v", err)
	}

	go sink.GarbageCollect()
	go sink.ProcessQueue()

	e := echo.New()
	e.HideBanner = true
	e.HidePort = true

	e.POST("/ingest", makeIngestRequestHandler(sink), metricsMiddleware)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	// Start server
	go func() {
		logger.Infof("Starting to listen on %s…", config.Listen)
		if err := e.Start(config.Listen); err != nil && err.Error() != "http: Server closed" {
			logger.Fatalf("Could not start server: %v", err)
		}
	}()

	// Wait for interrupt signal to gracefully shutdown the server with
	// a timeout of 10 seconds.
	quit := make(chan os.Signal, 5)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	shutdown(e, sink, logger)
}

func shutdown(server *echo.Echo, sink *sink, logger logrus.FieldLogger) {
	logger.Info("Received signal, shutting down…")

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	if err := server.Shutdown(ctx); err != nil {
		logger.Fatalf("Failed to shutdown HTTP server: %v", err)
	}

	logger.Info("HTTP server stopped.")

	logger.Info("Shutting down log processor…")
	sink.Close()
	logger.Info("Processor closed, exiting.")
}

func makeIngestRequestHandler(sink *sink) echo.HandlerFunc {
	return func(c echo.Context) error {
		req := c.Request()

		// check content type
		contentType := req.Header.Get(echo.HeaderContentType)
		if !strings.Contains(contentType, "json") {
			return c.String(http.StatusNotAcceptable, "Invalid Content-Type, ensure you send JSON payloads.")
		}

		// decode payload
		defer req.Body.Close()
		var payload Payload
		err := json.NewDecoder(req.Body).Decode(&payload)
		if err != nil {
			return c.String(http.StatusBadRequest, "Body could not be parsed as JSON")
		}

		// process payload
		received, ingested := sink.AddPayload(payload)
		recordsIngested.Add(float64(ingested))
		recordsReceived.Add(float64(received))

		if err != nil {
			log.Printf("Failed to store payload: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to store payload, check sink's logs.")
		}

		// done
		return c.NoContent(http.StatusOK)
	}
}

func makeLogger(config *Config) logrus.FieldLogger {
	logger := logrus.New()
	logger.SetLevel(logrus.InfoLevel)

	if config.Verbose {
		logger.SetLevel(logrus.DebugLevel)
	}

	return logger
}
