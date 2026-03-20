package main

import (
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp-contrib/middleware/idempotency"
	zconfig "github.com/alexferl/zerohttp/config"
	"github.com/alexferl/zerohttp/middleware"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Create Redis-backed idempotency store
	store := idempotency.NewRedisStore(client, "idempotency")

	app := zh.New()

	// Use idempotency middleware with Redis store
	app.Use(middleware.Idempotency(zconfig.IdempotencyConfig{
		Store: store,
	}))

	app.POST("/payments", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		// Process payment...
		log.Println("Processing payment...")
		return zh.R.JSON(w, http.StatusCreated, zh.M{
			"status":  "success",
			"message": "Payment processed",
		})
	}))

	log.Println("Server starting on :8080")
	log.Println("Test with:")
	log.Println(`curl -X POST http://localhost:8080/payments \`)
	log.Println(`  -H "Idempotency-Key: unique-key-123" \`)
	log.Println(`  -H "Content-Type: application/json" \`)
	log.Println(`  -d '{"amount": 100}'`)
	log.Fatal(app.Start())
}
