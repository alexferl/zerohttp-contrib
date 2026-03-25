package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// createTestStore creates a miniredis-based store for testing.
func createTestStore(t *testing.T) (*RedisStore, *miniredis.Miniredis) {
	t.Helper()
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})
	store := NewRedisStore(client, "test:")
	return store, s
}

func TestNewRedisStore(t *testing.T) {
	t.Run("with custom prefix", func(t *testing.T) {
		s := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: s.Addr()})
		store := NewRedisStore(client, "custom:")

		assert.Equal(t, "custom:", store.prefix)
		assert.NotNil(t, store.client)
	})

	t.Run("with empty prefix uses default", func(t *testing.T) {
		s := miniredis.RunT(t)
		client := redis.NewClient(&redis.Options{Addr: s.Addr()})
		store := NewRedisStore(client, "")

		assert.Equal(t, "jwt:", store.prefix)
	})
}

func TestRedisStore_tokenKey(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})

	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "with custom prefix",
			prefix:   "test:",
			key:      "user123:abc",
			expected: "test:token:user123:abc",
		},
		{
			name:     "with default prefix",
			prefix:   "",
			key:      "user123:abc",
			expected: "jwt:token:user123:abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRedisStore(client, tt.prefix)
			result := store.tokenKey(tt.key)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisStore_sessionKey(t *testing.T) {
	s := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{Addr: s.Addr()})

	tests := []struct {
		name     string
		prefix   string
		sid      string
		expected string
	}{
		{
			name:     "with custom prefix",
			prefix:   "test:",
			sid:      "session-abc",
			expected: "test:session:session-abc",
		},
		{
			name:     "with default prefix",
			prefix:   "",
			sid:      "session-abc",
			expected: "jwt:session:session-abc",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRedisStore(client, tt.prefix)
			result := store.sessionKey(tt.sid)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestRedisStore(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	t.Run("revoke and check token", func(t *testing.T) {
		err := store.RevokeToken(ctx, "token-123", 15*time.Minute)
		require.NoError(t, err)

		revoked, err := store.IsTokenRevoked(ctx, "token-123")
		require.NoError(t, err)
		assert.True(t, revoked)

		// Non-revoked token
		revoked, err = store.IsTokenRevoked(ctx, "token-456")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("revoke and check session", func(t *testing.T) {
		err := store.RevokeSession(ctx, "session-abc", 7*24*time.Hour)
		require.NoError(t, err)

		revoked, err := store.IsSessionRevoked(ctx, "session-abc")
		require.NoError(t, err)
		assert.True(t, revoked)

		// Non-revoked session
		revoked, err = store.IsSessionRevoked(ctx, "session-def")
		require.NoError(t, err)
		assert.False(t, revoked)
	})

	t.Run("context cancellation", func(t *testing.T) {
		cancelledCtx, cancel := context.WithCancel(context.Background())
		cancel()

		err := store.RevokeToken(cancelledCtx, "token", time.Minute)
		assert.ErrorIs(t, err, context.Canceled)

		_, err = store.IsTokenRevoked(cancelledCtx, "token")
		assert.ErrorIs(t, err, context.Canceled)
	})

	t.Run("close store", func(t *testing.T) {
		err := store.Close()
		require.NoError(t, err)
	})
}

func TestRedisStore_Client(t *testing.T) {
	store, _ := createTestStore(t)

	client := store.Client()
	assert.NotNil(t, client)
}

func TestRedisStore_Ping(t *testing.T) {
	store, _ := createTestStore(t)
	ctx := context.Background()

	err := store.Ping(ctx)
	require.NoError(t, err)
}
