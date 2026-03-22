package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	zconfig "github.com/alexferl/zerohttp/config"
)

// createTestStorage creates a miniredis-based storage for testing.
func createTestStorage(t *testing.T) (*RedisStorage, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	storage := NewRedisStorage(client, "test:")
	return storage, s
}

func createTestKeySet(t *testing.T) jwk.Set {
	rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
	key, err := jwk.Import(rawKey)
	require.NoError(t, err)

	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	require.NoError(t, err)

	return keySet
}

func TestNewTokenStore(t *testing.T) {
	keySet := createTestKeySet(t)
	storage, _ := createTestStorage(t)

	t.Run("valid configuration", func(t *testing.T) {
		cfg := Config{
			KeySet:  keySet,
			Storage: storage,
		}
		store := NewTokenStore(cfg)
		assert.NotNil(t, store)
	})

	t.Run("missing key set panics", func(t *testing.T) {
		cfg := Config{
			Storage: storage,
		}
		assert.Panics(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("empty key set panics", func(t *testing.T) {
		emptySet := jwk.NewSet()
		cfg := Config{
			KeySet:  emptySet,
			Storage: storage,
		}
		assert.Panics(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("missing storage panics", func(t *testing.T) {
		cfg := Config{
			KeySet: keySet,
		}
		assert.Panics(t, func() {
			NewTokenStore(cfg)
		})
	})
}

func TestTokenStore_Generate(t *testing.T) {
	keySet := createTestKeySet(t)
	storage, _ := createTestStorage(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   storage,
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}
	store := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("generate access token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate refresh token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := store.Generate(ctx, claims, zconfig.RefreshToken, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("nil claims returns empty map", func(t *testing.T) {
		token, err := store.Generate(ctx, nil, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with various claim types", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"aud": []string{"audience1", "audience2"},
			"iat": time.Now().Unix(),
			"nbf": time.Now().Unix(),
			"jti": "token-id-123",
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate and check claims preserved
		validated, err := store.Validate(ctx, token)
		require.NoError(t, err)
		m := validated.(map[string]any)
		assert.Equal(t, "user123", m["sub"])
	})

	t.Run("generate with aud as []interface{}", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"audience1", "audience2"},
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with iat as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": time.Now(),
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with nbf as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"nbf": time.Now(),
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with exp set in claims", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with iat and nbf as float64", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": float64(time.Now().Unix()),
			"nbf": float64(time.Now().Unix()),
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestTokenStore_Validate(t *testing.T) {
	keySet := createTestKeySet(t)
	storage, _ := createTestStorage(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   storage,
	}
	store := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("validate valid token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := store.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		validatedClaims, err := store.Validate(ctx, token)
		require.NoError(t, err)
		assert.NotNil(t, validatedClaims)

		// Check claims were preserved
		m, ok := validatedClaims.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "user123", m["sub"])
		assert.Equal(t, "session-abc", m["sid"])
	})

	t.Run("validate invalid token", func(t *testing.T) {
		_, err := store.Validate(ctx, "invalid.token.here")
		assert.Error(t, err)
	})

	t.Run("validate with issuer", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Storage:        storage,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		storeWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "expected-issuer",
		}

		token, err := storeWithIssuer.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithIssuer.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with wrong issuer fails", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Storage:        storage,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		storeWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "wrong-issuer",
		}

		token, err := storeWithIssuer.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithIssuer.Validate(ctx, token)
		assert.Error(t, err)
	})

	t.Run("validate with audience", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          storage,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		storeWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "expected-audience",
		}

		token, err := storeWithAudience.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with wrong audience fails", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          storage,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		storeWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "wrong-audience",
		}

		token, err := storeWithAudience.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithAudience.Validate(ctx, token)
		assert.Error(t, err)
	})

	t.Run("validate with audience array", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          storage,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		storeWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []string{"other-audience", "expected-audience"},
		}

		token, err := storeWithAudience.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with audience []interface{}", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          storage,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		storeWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"expected-audience"},
		}

		token, err := storeWithAudience.Generate(ctx, claims, zconfig.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = storeWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})
}

func TestTokenStore_Revoke(t *testing.T) {
	keySet := createTestKeySet(t)
	storage, _ := createTestStorage(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   storage,
	}
	store := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("revoke token and session", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		err := store.Revoke(ctx, claims)
		require.NoError(t, err)

		// Check session is revoked
		revoked, err := store.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("is revoked returns false for non-revoked token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user456",
			"sid": "session-def",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		revoked, err := store.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("revoke token without session", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user789",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		err := store.Revoke(ctx, claims)
		require.NoError(t, err)

		revoked, err := store.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("is revoked with missing exp returns false", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user-no-exp",
		}

		revoked, err := store.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.False(t, revoked)
	})
}

func TestDefaultTokenKeyFunc(t *testing.T) {
	t.Run("generates key from sub and exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": int64(1234567890),
		}

		key := defaultTokenKeyFunc(claims)
		assert.Contains(t, key, "user123")
	})

	t.Run("handles float64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": float64(1234567890),
		}

		key := defaultTokenKeyFunc(claims)
		assert.Contains(t, key, "user123")
	})

	t.Run("formats key correctly", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": int64(1234567890),
		}

		key := defaultTokenKeyFunc(claims)
		assert.Equal(t, "user123:1234567890", key)
	})

	t.Run("handles empty sub", func(t *testing.T) {
		claims := map[string]any{
			"exp": int64(1234567890),
		}

		key := defaultTokenKeyFunc(claims)
		assert.Equal(t, ":1234567890", key)
	})

	t.Run("handles zero exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
		}

		key := defaultTokenKeyFunc(claims)
		assert.Equal(t, "user123:0", key)
	})
}

func TestCalculateTTL(t *testing.T) {
	keySet := createTestKeySet(t)
	storage, _ := createTestStorage(t)
	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   storage,
	}
	store := NewTokenStore(cfg)

	t.Run("calculateTTL with int64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}
		ttl := store.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with float64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(15 * time.Minute).Unix()),
		}
		ttl := store.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with time.Time exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute),
		}
		ttl := store.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with expired token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(-15 * time.Minute).Unix(),
		}
		ttl := store.calculateTTL(claims)
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("calculateTTL with no exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
		}
		ttl := store.calculateTTL(claims)
		assert.Equal(t, time.Duration(0), ttl)
	})
}

func TestNormalizeClaims(t *testing.T) {
	t.Run("map[string]any", func(t *testing.T) {
		claims := map[string]any{"sub": "user123"}
		result, err := normalizeClaims(claims)
		require.NoError(t, err)
		assert.Equal(t, claims, result)
	})

	t.Run("nil claims", func(t *testing.T) {
		result, err := normalizeClaims(nil)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Empty(t, result)
	})

	t.Run("unsupported type", func(t *testing.T) {
		claims := "invalid"
		_, err := normalizeClaims(claims)
		assert.Error(t, err)
	})
}

func TestRedisStorage(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	t.Run("revoke and check token", func(t *testing.T) {
		err := storage.RevokeToken(ctx, "token-123", 15*time.Minute)
		require.NoError(t, err)

		revoked, err := storage.IsTokenRevoked(ctx, "token-123")
		require.NoError(t, err)
		assert.True(t, revoked)

		// Non-revoked token
		revoked, err = storage.IsTokenRevoked(ctx, "token-456")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("revoke and check session", func(t *testing.T) {
		err := storage.RevokeSession(ctx, "session-abc", 7*24*time.Hour)
		require.NoError(t, err)

		revoked, err := storage.IsSessionRevoked(ctx, "session-abc")
		require.NoError(t, err)
		assert.True(t, revoked)

		// Non-revoked session
		revoked, err = storage.IsSessionRevoked(ctx, "session-def")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := storage.RevokeToken(cancelledCtx, "token", time.Minute)
		assert.ErrorIs(t, err, context.Canceled)

		_, err = storage.IsTokenRevoked(cancelledCtx, "token")
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("close storage", func(t *testing.T) {
		err := storage.Close()
		require.NoError(t, err)
	})
}

func TestRedisStorage_Client(t *testing.T) {
	storage, _ := createTestStorage(t)

	client := storage.Client()
	assert.NotNil(t, client)
}

func TestRedisStorage_Ping(t *testing.T) {
	storage, _ := createTestStorage(t)
	ctx := context.Background()

	err := storage.Ping(ctx)
	require.NoError(t, err)
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.NotNil(t, cfg.TokenKeyFunc)
	assert.NotNil(t, cfg.KeySelector)
	assert.Equal(t, "HS256", cfg.Algorithm.String())
	assert.True(t, cfg.ValidateExpiration)
	assert.True(t, cfg.ValidateNotBefore)
}
