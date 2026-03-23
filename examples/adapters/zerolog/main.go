package main

import (
	"log"
	"net/http"
	"os"

	zh "github.com/alexferl/zerohttp"
	zclog "github.com/alexferl/zerohttp-contrib/adapters/zerolog"
	"github.com/rs/zerolog"
)

func main() {
	// Create a zerolog logger with console output (human-readable)
	zl := zerolog.New(zerolog.ConsoleWriter{Out: os.Stdout}).
		Level(zerolog.InfoLevel).
		With().
		Timestamp().
		Caller().
		Logger()

	// Wrap it for zerohttp
	logger := zclog.New(zl)

	app := zh.New(zh.Config{
		Logger: logger,
	})

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		app.Logger().Info("handling request")
		return zh.R.JSON(w, http.StatusOK, zh.M{"message": "Hello, World!"})
	}))

	app.GET("/error", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		app.Logger().Error("something went wrong")
		return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "internal error"})
	}))

	log.Fatal(app.Start())
}
