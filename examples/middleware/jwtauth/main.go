package main

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp-contrib/middleware/jwtauth"
	zjwtauth "github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/redis/go-redis/v9"
)

func main() {
	app := zh.New()

	// Create a symmetric key for HS256
	// In production, load this from a secure location
	rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
	key, err := jwk.Import(rawKey)
	if err != nil {
		log.Fatalf("failed to import key: %s", err)
	}

	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	if err != nil {
		log.Fatalf("failed to add key: %s", err)
	}

	// Connect to Redis for token revocation storage
	redisClient := redis.NewClient(&redis.Options{
		Addr: "localhost:6379",
	})

	// Check Redis connection
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	if err := redisClient.Ping(ctx).Err(); err != nil {
		log.Fatalf("failed to connect to Redis: %s", err)
	}

	// Create TokenStore using lestrrat-go/jwx with Redis storage
	cfg := jwtauth.Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   jwtauth.NewRedisStorage(redisClient, "jwt"),
	}
	tokenStore := jwtauth.NewTokenStore(cfg)

	jwtCfg := zjwtauth.Config{
		Store:           tokenStore,
		RequiredClaims:  []string{"sub"},
		ExcludedPaths:   []string{"/login"},
		AccessTokenTTL:  15 * time.Minute,
		RefreshTokenTTL: 7 * 24 * time.Hour,
	}

	// Public login endpoint
	app.POST("/login", loginHandler(jwtCfg))

	// Refresh token endpoint
	app.POST("/auth/refresh", zjwtauth.RefreshTokenHandler(jwtCfg))

	// Logout endpoint
	app.POST("/auth/logout", zjwtauth.LogoutTokenHandler(jwtCfg))

	// Protected endpoints
	app.Use(zjwtauth.New(jwtCfg))

	app.GET("/api/profile", zh.HandlerFunc(profileHandler))

	log.Fatal(app.Start())
}

func loginHandler(cfg zjwtauth.Config) zh.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) error {
		var req struct {
			Username string `json:"username"`
			Password string `json:"password"`
		}

		if err := zh.B.JSON(r.Body, &req); err != nil {
			return zh.R.JSON(w, http.StatusBadRequest, zh.M{"error": "invalid request"})
		}

		// Demo credentials
		if req.Username != "alice" || req.Password != "secret" {
			return zh.R.JSON(w, http.StatusUnauthorized, zh.M{"error": "invalid credentials"})
		}

		// Generate a session ID that links access and refresh tokens
		sessionID := fmt.Sprintf("%s_%d", req.Username, time.Now().UnixNano())

		claims := map[string]any{
			"sub":   req.Username,
			"scope": "read write",
			"sid":   sessionID,
		}

		accessToken, err := zjwtauth.GenerateAccessToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		refreshToken, err := zjwtauth.GenerateRefreshToken(r, claims, cfg)
		if err != nil {
			return zh.R.JSON(w, http.StatusInternalServerError, zh.M{"error": "failed to generate token"})
		}

		return zh.R.JSON(w, http.StatusOK, zh.M{
			"access_token":  accessToken,
			"refresh_token": refreshToken,
			"token_type":    "Bearer",
			"expires_in":    int(cfg.AccessTokenTTL.Seconds()),
		})
	}
}

func profileHandler(w http.ResponseWriter, r *http.Request) error {
	jwtClaims := zjwtauth.GetClaims(r)

	return zh.R.JSON(w, http.StatusOK, zh.M{
		"subject": jwtClaims.Subject(),
		"scopes":  jwtClaims.Scopes(),
		"message": "Hello from lestrrat-go/jwx v3 JWT auth",
	})
}
