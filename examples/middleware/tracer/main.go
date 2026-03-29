package main

import (
	"context"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp-contrib/middleware/tracer"
	ztracer "github.com/alexferl/zerohttp/middleware/tracer"
)

func main() {
	ctx := context.Background()

	// Setup tracer with default OTLP HTTP exporter
	tracerImpl, shutdown, err := tracer.NewHTTPDefault(ctx, "zerohttp-example", "localhost:4318", true)
	if err != nil {
		log.Fatalf("Failed to create tracer: %v", err)
	}
	defer shutdown()

	app := zh.New(zh.Config{Tracer: ztracer.Config{TracerField: tracerImpl}})
	app.Use(ztracer.New(tracerImpl))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusOK, map[string]string{"message": "Hello!"})
	}))

	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zh.R.JSON(w, http.StatusInternalServerError, map[string]string{"error": "Oops"})
	}))

	log.Fatal(app.Start())
}
