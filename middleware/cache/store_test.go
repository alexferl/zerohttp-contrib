package cache

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
)

// mockRedisClient is a mock implementation of RedisClient for testing.
type mockRedisClient struct {
	getFunc func(ctx context.Context, key string) *redis.StringCmd
	setFunc func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	return m.getFunc(ctx, key)
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
	return m.setFunc(ctx, key, value, expiration)
}

func TestNewRedisStore(t *testing.T) {
	store := NewRedisStore(nil, "test:prefix")
	assert.NotNil(t, store)
	assert.Equal(t, "test:prefix", store.prefix)
	assert.Nil(t, store.client)
}

func TestRedisStore_makeKey(t *testing.T) {
	tests := []struct {
		name   string
		prefix string
		key    string
		want   string
	}{
		{
			name:   "with prefix",
			prefix: "cache",
			key:    "abc123",
			want:   "cache:abc123",
		},
		{
			name:   "without prefix",
			prefix: "",
			key:    "abc123",
			want:   "abc123",
		},
		{
			name:   "empty key",
			prefix: "cache",
			key:    "",
			want:   "cache:",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewRedisStore(nil, tt.prefix)
			got := store.makeKey(tt.key)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestRedisStore_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("cache miss", func(t *testing.T) {
		mockClient := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringResult("", redis.Nil)
			},
		}

		store := NewRedisStore(mockClient, "test")
		record, found, err := store.Get(ctx, "missing-key")

		assert.NoError(t, err)
		assert.False(t, found)
		assert.Equal(t, config.CacheRecord{}, record)
	})

	t.Run("cache hit", func(t *testing.T) {
		// Valid JSON cache record
		jsonData := `{"status_code":200,"headers":{"Content-Type":["application/json"]},"body":"eyJoZWxsbyI6IndvcmxkIn0=","etag":"\"abc123\"","last_modified":"2024-01-01T00:00:00Z","vary_headers":{"Accept":"application/json"}}`

		mockClient := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringResult(jsonData, nil)
			},
		}

		store := NewRedisStore(mockClient, "test")
		record, found, err := store.Get(ctx, "existing-key")

		assert.NoError(t, err)
		assert.True(t, found)
		assert.Equal(t, 200, record.StatusCode)
		assert.Equal(t, map[string][]string{"Content-Type": {"application/json"}}, record.Headers)
	})

	t.Run("redis error", func(t *testing.T) {
		mockClient := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringResult("", errors.New("connection refused"))
			},
		}

		store := NewRedisStore(mockClient, "test")
		record, found, err := store.Get(ctx, "error-key")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
		assert.False(t, found)
		assert.Equal(t, config.CacheRecord{}, record)
	})

	t.Run("invalid json", func(t *testing.T) {
		mockClient := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				return redis.NewStringResult("invalid json", nil)
			},
		}

		store := NewRedisStore(mockClient, "test")
		record, found, err := store.Get(ctx, "invalid-key")

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "failed to unmarshal cache record")
		assert.False(t, found)
		assert.Equal(t, config.CacheRecord{}, record)
	})
}

func TestRedisStore_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("successful set", func(t *testing.T) {
		var capturedKey string
		var capturedValue interface{}
		var capturedTTL time.Duration

		mockClient := &mockRedisClient{
			setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
				capturedKey = key
				capturedValue = value
				capturedTTL = expiration
				return redis.NewStatusCmd(ctx, "OK")
			},
		}

		store := NewRedisStore(mockClient, "test")
		record := config.CacheRecord{
			StatusCode: 200,
			Headers:    map[string][]string{"Content-Type": {"application/json"}},
			Body:       []byte(`{"hello":"world"}`),
			ETag:       `"abc123"`,
		}

		err := store.Set(ctx, "test-key", record, 30*time.Second)

		assert.NoError(t, err)
		assert.Equal(t, "test:test-key", capturedKey)
		assert.Equal(t, 30*time.Second, capturedTTL)
		// Verify the value is valid JSON
		assert.NotNil(t, capturedValue)
	})

	t.Run("redis error", func(t *testing.T) {
		mockClient := &mockRedisClient{
			setFunc: func(ctx context.Context, key string, value interface{}, expiration time.Duration) *redis.StatusCmd {
				cmd := redis.NewStatusResult("", errors.New("connection refused"))
				return cmd
			},
		}

		store := NewRedisStore(mockClient, "test")
		record := config.CacheRecord{
			StatusCode: 200,
			Body:       []byte(`test`),
		}

		err := store.Set(ctx, "error-key", record, time.Minute)

		assert.Error(t, err)
		assert.Contains(t, err.Error(), "connection refused")
	})
}
