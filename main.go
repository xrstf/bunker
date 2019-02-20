package main

import (
	"encoding/json"
	"log"
	"net/http"
	"strings"

	"github.com/labstack/echo"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

type Config struct {
	Target  string
	Pattern string
}

func main() {
	config := Config{
		Target:  "records",
		Pattern: "%date%/%kubernetes_namespace_name%/%kubernetes_pod_name%-%kubernetes_container_name%.json",
	}

	e := echo.New()
	e.POST("/ingest", makeIngestRequestHandler(config), metricsMiddleware)
	e.GET("/metrics", echo.WrapHandler(promhttp.Handler()))

	e.Logger.Fatal(e.Start(":1323"))
}

func makeIngestRequestHandler(config Config) echo.HandlerFunc {
	sink := NewSink(config)

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
		processed, err := sink.StorePayload(payload)
		recordsIngested.Add(float64(processed))

		if err != nil {
			log.Printf("Failed to store payload: %v", err)
			return c.String(http.StatusInternalServerError, "Failed to store payload, check sink's logs.")
		}

		// done
		return c.NoContent(http.StatusOK)
	}
}
