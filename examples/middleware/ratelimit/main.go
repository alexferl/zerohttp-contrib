package main

import (
	"context"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	contribRatelimit "github.com/alexferl/zerohttp-contrib/middleware/ratelimit"
	zratelimit "github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/redis/go-redis/v9"
)

func main() {
	// Create Redis client
	client := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Test Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := client.Ping(ctx).Err(); err != nil {
		log.Fatalf("Failed to connect to Redis: %v\nMake sure Redis is running: docker run -d --name redis -p 6379:6379 redis:7-alpine", err)
	}

	// Create Redis-backed rate limit store using the contrib package
	rate := 10
	window := time.Minute
	store := contribRatelimit.NewRedisStore(client, zratelimit.SlidingWindow, window, rate)

	// Configure the server with Redis store
	app := zh.New()
	app.Use(zratelimit.New(zratelimit.Config{
		Store:          store,
		Rate:           rate,
		Window:         window,
		IncludeHeaders: true,
	}))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		limit := w.Header().Get("X-RateLimit-Limit")
		remaining := w.Header().Get("X-RateLimit-Remaining")
		reset := w.Header().Get("X-RateLimit-Reset")

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"message":   "Hello with Redis rate limiting!",
			"limit":     limit,
			"remaining": remaining,
			"reset":     reset,
		})
	}))

	log.Fatal(app.Start())
}
