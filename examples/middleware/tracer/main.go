package main

import (
	"context"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp-contrib/middleware/tracer"
	"github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
)

func main() {
	ctx := context.Background()

	// Setup tracer with default OTLP HTTP exporter
	tracerImpl, shutdown, err := tracer.NewDefault(ctx, "zerohttp-example", "localhost:4318")
	if err != nil {
		log.Fatalf("Failed to create tracer: %v", err)
	}
	defer shutdown()

	app := zh.New(config.Config{Tracer: config.TracerConfig{TracerField: tracerImpl}})
	app.Use(middleware.Tracer(tracerImpl))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{"message": "Hello!"})
	}))

	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusInternalServerError, map[string]string{"error": "Oops"})
	}))

	log.Fatal(app.Start())
}
