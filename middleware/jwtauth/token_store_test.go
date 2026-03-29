package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"

	"github.com/alexferl/zerohttp/middleware/jwtauth"
	"github.com/alexferl/zerohttp/storage"
	"github.com/alexferl/zerohttp/zhtest"
)

// testStorage is a simple in-memory implementation of storage.Storage for testing.
type testStorage struct {
	data map[string][]byte
}

func newTestStorage() *testStorage {
	return &testStorage{data: make(map[string][]byte)}
}

func (m *testStorage) Get(_ context.Context, key string) ([]byte, bool, error) {
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *testStorage) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	m.data[key] = val
	return nil
}

func (m *testStorage) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *testStorage) Close() error {
	return nil
}

func createTestStorage() storage.Storage {
	return newTestStorage()
}

func createTestKeySet(t *testing.T) jwk.Set {
	rawKey := []byte("your-secret-key-at-least-32-bytes-long!")
	key, err := jwk.Import(rawKey)
	zhtest.AssertNoError(t, err)

	keySet := jwk.NewSet()
	err = keySet.AddKey(key)
	zhtest.AssertNoError(t, err)

	return keySet
}

func TestNewTokenStore(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()

	t.Run("valid configuration", func(t *testing.T) {
		cfg := Config{
			KeySet:  keySet,
			Storage: store,
		}
		tokenStore := NewTokenStore(cfg)
		zhtest.AssertNotNil(t, tokenStore)
	})

	t.Run("missing key set panics", func(t *testing.T) {
		cfg := Config{
			Storage: store,
		}
		zhtest.AssertPanic(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("empty key set panics", func(t *testing.T) {
		emptySet := jwk.NewSet()
		cfg := Config{
			KeySet:  emptySet,
			Storage: store,
		}
		zhtest.AssertPanic(t, func() {
			NewTokenStore(cfg)
		})
	})

	t.Run("missing store panics", func(t *testing.T) {
		cfg := Config{
			KeySet: keySet,
		}
		zhtest.AssertPanic(t, func() {
			NewTokenStore(cfg)
		})
	})
}

func TestTokenStore_Generate(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   store,
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
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("generate refresh token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.RefreshToken, 7*24*time.Hour)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("nil claims returns empty map", func(t *testing.T) {
		token, err := tokenStore.Generate(ctx, nil, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
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
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)

		// Validate and check claims preserved
		validated, err := tokenStore.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
		m := validated.(map[string]any)
		zhtest.AssertEqual(t, "user123", m["sub"])
	})

	t.Run("generate with aud as []interface{}", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"audience1", "audience2"},
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("generate with iat as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": time.Now(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("generate with nbf as time.Time", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"nbf": time.Now(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("generate with exp set in claims", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(time.Hour).Unix(),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})

	t.Run("generate with iat and nbf as float64", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"iat": float64(time.Now().Unix()),
			"nbf": float64(time.Now().Unix()),
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotEmpty(t, token)
	})
}

func TestTokenStore_Validate(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   store,
	}
	tokenStore := NewTokenStore(cfg)

	ctx := context.Background()

	t.Run("validate valid token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"sid": "session-abc",
		}

		token, err := tokenStore.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		validatedClaims, err := tokenStore.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotNil(t, validatedClaims)

		// Check claims were preserved
		m, ok := validatedClaims.(map[string]any)
		zhtest.AssertTrue(t, ok)
		zhtest.AssertEqual(t, "user123", m["sub"])
		zhtest.AssertEqual(t, "session-abc", m["sid"])
	})

	t.Run("validate invalid token", func(t *testing.T) {
		_, err := tokenStore.Validate(ctx, "invalid.token.here")
		zhtest.AssertError(t, err)
	})

	t.Run("validate with issuer", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Storage:        store,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		tokenStoreWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "expected-issuer",
		}

		token, err := tokenStoreWithIssuer.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithIssuer.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
	})

	t.Run("validate with wrong issuer fails", func(t *testing.T) {
		cfgWithIssuer := Config{
			KeySet:         keySet,
			Algorithm:      jwa.HS256(),
			Storage:        store,
			Issuer:         "expected-issuer",
			ValidateIssuer: true,
		}
		tokenStoreWithIssuer := NewTokenStore(cfgWithIssuer)

		claims := map[string]any{
			"sub": "user123",
			"iss": "wrong-issuer",
		}

		token, err := tokenStoreWithIssuer.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithIssuer.Validate(ctx, token)
		zhtest.AssertError(t, err)
	})

	t.Run("validate with audience", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "expected-audience",
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
	})

	t.Run("validate with wrong audience fails", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": "wrong-audience",
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		zhtest.AssertError(t, err)
	})

	t.Run("validate with audience array", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []string{"other-audience", "expected-audience"},
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
	})

	t.Run("validate with audience []interface{}", func(t *testing.T) {
		cfgWithAudience := Config{
			KeySet:           keySet,
			Algorithm:        jwa.HS256(),
			Storage:          store,
			Audience:         "expected-audience",
			ValidateAudience: true,
		}
		tokenStoreWithAudience := NewTokenStore(cfgWithAudience)

		claims := map[string]any{
			"sub": "user123",
			"aud": []interface{}{"expected-audience"},
		}

		token, err := tokenStoreWithAudience.Generate(ctx, claims, jwtauth.AccessToken, 15*time.Minute)
		zhtest.AssertNoError(t, err)

		_, err = tokenStoreWithAudience.Validate(ctx, token)
		zhtest.AssertNoError(t, err)
	})
}

func TestTokenStore_Revoke(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   store,
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
		zhtest.AssertNoError(t, err)

		// Check session is revoked
		revoked, err := tokenStore.IsRevoked(ctx, claims)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, revoked)
	})

	t.Run("is revoked returns false for non-revoked token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user456",
			"sid": "session-def",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, revoked)
	})

	t.Run("revoke token without session", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user789",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}

		err := tokenStore.Revoke(ctx, claims)
		zhtest.AssertNoError(t, err)

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, revoked)
	})

	t.Run("is revoked with missing exp returns false", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user-no-exp",
		}

		revoked, err := tokenStore.IsRevoked(ctx, claims)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, revoked)
	})
}

func TestCalculateTTL(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()
	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   store,
	}
	tokenStore := NewTokenStore(cfg)

	t.Run("calculateTTL with int64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute).Unix(),
		}
		ttl := tokenStore.calculateTTL(claims)
		zhtest.AssertTrue(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with float64 exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": float64(time.Now().Add(15 * time.Minute).Unix()),
		}
		ttl := tokenStore.calculateTTL(claims)
		zhtest.AssertTrue(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with time.Time exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(15 * time.Minute),
		}
		ttl := tokenStore.calculateTTL(claims)
		zhtest.AssertTrue(t, ttl > 14*time.Minute && ttl <= 15*time.Minute)
	})

	t.Run("calculateTTL with expired token", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
			"exp": time.Now().Add(-15 * time.Minute).Unix(),
		}
		ttl := tokenStore.calculateTTL(claims)
		zhtest.AssertEqual(t, time.Duration(0), ttl)
	})

	t.Run("calculateTTL with no exp", func(t *testing.T) {
		claims := map[string]any{
			"sub": "user123",
		}
		ttl := tokenStore.calculateTTL(claims)
		zhtest.AssertEqual(t, time.Duration(0), ttl)
	})
}

func TestNormalizeClaims(t *testing.T) {
	t.Run("map[string]any", func(t *testing.T) {
		claims := map[string]any{"sub": "user123"}
		result, err := normalizeClaims(claims)
		zhtest.AssertNoError(t, err)
		zhtest.AssertDeepEqual(t, claims, result)
	})

	t.Run("nil claims", func(t *testing.T) {
		result, err := normalizeClaims(nil)
		zhtest.AssertNoError(t, err)
		zhtest.AssertNotNil(t, result)
		zhtest.AssertEmpty(t, result)
	})

	t.Run("unsupported type", func(t *testing.T) {
		claims := "invalid"
		_, err := normalizeClaims(claims)
		zhtest.AssertError(t, err)
	})
}

func TestTokenStore_Close(t *testing.T) {
	keySet := createTestKeySet(t)
	store := createTestStorage()

	cfg := Config{
		KeySet:    keySet,
		Algorithm: jwa.HS256(),
		Storage:   store,
	}
	tokenStore := NewTokenStore(cfg)

	err := tokenStore.Close()
	zhtest.AssertNoError(t, err)
}
