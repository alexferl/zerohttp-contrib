package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
)

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
	store, _ := createTestStore(t)

	t.Run("valid configuration", func(t *testing.T) {
		cfg := Config{
			KeySet: keySet,
			Store:  store,
		}
		tokenStore := NewTokenStore(cfg)
		assert.NotNil(t, tokenStore)
	})

	t.Run("missing key set panics", func(t *testing.T) {
		cfg := Config{
			Store: store,
		}
		assert.Panics(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("empty key set panics", func(t *testing.T) {
		emptySet := jwk.NewSet()
		cfg := Config{
			KeySet: emptySet,
			Store:  store,
		}
		assert.Panics(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("missing store panics", func(t *testing.T) {
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
	store, _ := createTestStore(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Store:     store,
		Issuer:    "test-issuer",
		Audience:  "test-audience",
	}
	tokenStore := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("generate access token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate refresh token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.RefreshToken, 7*24*time.Hour)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("nil claims returns empty map", func(t *testing.T) {
		token, err := tokenStore.Generate(ctx, nil, jwtauth.AccessToken, 15*time.Minute)
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

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)

		// Validate and check claims preserved
		validated, err := tokenStore.Validate(ctx, token)
		require.NoError(t, err)
		m := validated.(map[string]any)
		assert.Equal(t, "user123", m["sub"])
	})

	t.Run("generate with aud as []interface{}", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"audience1", "audience2"},
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with iat as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": time.Now(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with nbf as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"nbf": time.Now(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with exp set in claims", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})

	t.Run("generate with iat and nbf as float64", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": float64(time.Now().Unix()),
			"nbf": float64(time.Now().Unix()),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)
		assert.NotEmpty(t, token)
	})
}

func TestTokenStore_Validate(t *testing.T) {
	keySet := createTestKeySet(t)
	store, _ := createTestStore(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Store:     store,
	}
	tokenStore := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("validate valid token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		validatedClaims, err := tokenStore.Validate(ctx, token)
		require.NoError(t, err)
		assert.NotNil(t, validatedClaims)

		// Check claims were preserved
		m, ok := validatedClaims.(map[string]any)
		require.True(t, ok)
		assert.Equal(t, "user123", m["sub"])
		assert.Equal(t, "session-abc", m["sid"])
	})

	t.Run("validate invalid token", func(t *testing.T) {
		_, err := tokenStore.Validate(ctx, "invalid.token.here")
		assert.Error(t, err)
	})

	t.Run("validate with issuer", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Store:          store,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		tokenStoreWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "expected-issuer",
		}

		token, err := tokenStoreWithIssuer.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithIssuer.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with wrong issuer fails", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Store:          store,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		tokenStoreWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "wrong-issuer",
		}

		token, err := tokenStoreWithIssuer.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithIssuer.Validate(ctx, token)
		assert.Error(t, err)
	})

	t.Run("validate with audience", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Store:            store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "expected-audience",
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with wrong audience fails", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Store:            store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "wrong-audience",
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		assert.Error(t, err)
	})

	t.Run("validate with audience array", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Store:            store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []string{"other-audience", "expected-audience"},
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})

	t.Run("validate with audience []interface{}", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Store:            store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"expected-audience"},
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		require.NoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		require.NoError(t, err)
	})
}

func TestTokenStore_Revoke(t *testing.T) {
	keySet := createTestKeySet(t)
	store, _ := createTestStore(t)

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Store:     store,
	}
	tokenStore := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("revoke token and session", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		err := tokenStore.Revoke(ctx, claims)
		require.NoError(t, err)

		// Check session is revoked
		revoked, err := tokenStore.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("is revoked returns false for non-revoked token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user456",
			"sid": "session-def",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("revoke token without session", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user789",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		err := tokenStore.Revoke(ctx, claims)
		require.NoError(t, err)

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.True(t, revoked)
	})

	t.Run("is revoked with missing exp returns false", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user-no-exp",
		}

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		require.NoError(t, err)
		assert.False(t, revoked)
	})
}

func TestCalculateTTL(t *testing.T) {
	keySet := createTestKeySet(t)
	store, _ := createTestStore(t)
	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Store:     store,
	}
	tokenStore := NewTokenStore(cfg)

	t.Run("calculateTTL with int64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}
		ttl := tokenStore.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with float64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(15 * time.Minute).Unix()),
		}
		ttl := tokenStore.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with time.Time exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute),
		}
		ttl := tokenStore.calculateTTL(claims)
		assert.True(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with expired token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(-15 * time.Minute).Unix(),
		}
		ttl := tokenStore.calculateTTL(claims)
		assert.Equal(t, time.Duration(0), ttl)
	})

	t.Run("calculateTTL with no exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
		}
		ttl := tokenStore.calculateTTL(claims)
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
