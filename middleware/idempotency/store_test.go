package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/config"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func setupTestRedis(t *testing.T) (*miniredis.Miniredis, RedisClient) {
	mr := miniredis.RunT(t)
	client := redis.NewClient(&redis.Options{
		Addr: mr.Addr(),
	})
	return mr, client
}

func TestNewRedisStore(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisStore(client, "test")

	assert.NotNil(t, store)
	assert.Equal(t, "test", store.keyPrefix)
	assert.Equal(t, 30*time.Second, store.lockTTL)
}

func TestNewRedisStoreWithLockTTL(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisStoreWithLockTTL(client, "test", 10*time.Second)

	assert.NotNil(t, store)
	assert.Equal(t, "test", store.keyPrefix)
	assert.Equal(t, 10*time.Second, store.lockTTL)
}

func TestRedisStore_SetAndGet(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	record := config.IdempotencyRecord{
		StatusCode: 200,
		Headers:    []string{"Content-Type", "application/json"},
		Body:       []byte(`{"success":true}`),
		CreatedAt:  time.Now().UTC(),
	}

	// Set the record
	err := store.Set(ctx, "key1", record, time.Hour)
	require.NoError(t, err)

	// Get the record
	retrieved, found, err := store.Get(ctx, "key1")
	require.NoError(t, err)
	assert.True(t, found)
	assert.Equal(t, record.StatusCode, retrieved.StatusCode)
	assert.Equal(t, record.Headers, retrieved.Headers)
	assert.Equal(t, record.Body, retrieved.Body)
}

func TestRedisStore_Get_NotFound(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	retrieved, found, err := store.Get(ctx, "nonexistent")
	require.NoError(t, err)
	assert.False(t, found)
	assert.Empty(t, retrieved.StatusCode)
}

func TestRedisStore_Set_WithTTL(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	record := config.IdempotencyRecord{
		StatusCode: 201,
		Body:       []byte(`created`),
	}

	// Set with short TTL
	err := store.Set(ctx, "key2", record, time.Minute)
	require.NoError(t, err)

	// Should exist immediately
	_, found, err := store.Get(ctx, "key2")
	require.NoError(t, err)
	assert.True(t, found)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Minute)

	// Should not exist after TTL
	_, found, err = store.Get(ctx, "key2")
	require.NoError(t, err)
	assert.False(t, found)
}

func TestRedisStore_Lock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// First lock should succeed
	acquired, err := store.Lock(ctx, "resource1")
	require.NoError(t, err)
	assert.True(t, acquired)

	// Second lock should fail (already locked)
	acquired, err = store.Lock(ctx, "resource1")
	require.NoError(t, err)
	assert.False(t, acquired)

	// Different resource should succeed
	acquired, err = store.Lock(ctx, "resource2")
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestRedisStore_Unlock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// Acquire lock
	acquired, err := store.Lock(ctx, "resource")
	require.NoError(t, err)
	assert.True(t, acquired)

	// Unlock
	err = store.Unlock(ctx, "resource")
	require.NoError(t, err)

	// Should be able to lock again
	acquired, err = store.Lock(ctx, "resource")
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestRedisStore_LockTTL(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	// Create store with short lock TTL
	store := NewRedisStoreWithLockTTL(client, "idemp", time.Minute)
	ctx := context.Background()

	// Acquire lock
	acquired, err := store.Lock(ctx, "resource")
	require.NoError(t, err)
	assert.True(t, acquired)

	// Should not be able to lock (still held)
	acquired, err = store.Lock(ctx, "resource")
	require.NoError(t, err)
	assert.False(t, acquired)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Minute)

	// Should be able to acquire lock now
	acquired, err = store.Lock(ctx, "resource")
	require.NoError(t, err)
	assert.True(t, acquired)
}

func TestRedisStore_KeyPrefix(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "myprefix")
	ctx := context.Background()

	record := config.IdempotencyRecord{
		StatusCode: 200,
		Body:       []byte(`ok`),
	}

	err := store.Set(ctx, "mykey", record, time.Hour)
	require.NoError(t, err)

	// Verify key exists with prefix in Redis
	exists := mr.Exists("myprefix:mykey")
	assert.True(t, exists)
}

func TestRedisStore_NoPrefix(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "")
	ctx := context.Background()

	record := config.IdempotencyRecord{
		StatusCode: 200,
		Body:       []byte(`ok`),
	}

	err := store.Set(ctx, "mykey", record, time.Hour)
	require.NoError(t, err)

	// Verify key exists without prefix in Redis
	exists := mr.Exists("mykey")
	assert.True(t, exists)
}

func TestRedisStore_Get_InvalidData(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// Set invalid JSON directly in Redis
	err := client.Set(ctx, "idemp:badkey", "not-json", time.Hour).Err()
	require.NoError(t, err)

	// Get should return error for invalid data
	_, found, err := store.Get(ctx, "badkey")
	assert.Error(t, err)
	assert.False(t, found)
}

func TestRedisStore_ConcurrentLocks(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// First goroutine acquires lock
	acquired1, err := store.Lock(ctx, "concurrent")
	require.NoError(t, err)
	assert.True(t, acquired1)

	// Multiple concurrent lock attempts should all fail
	for i := 0; i < 5; i++ {
		acquired, err := store.Lock(ctx, "concurrent")
		require.NoError(t, err)
		assert.False(t, acquired, "attempt %d should fail", i)
	}

	// After unlock, one should succeed
	err = store.Unlock(ctx, "concurrent")
	require.NoError(t, err)

	acquired2, err := store.Lock(ctx, "concurrent")
	require.NoError(t, err)
	assert.True(t, acquired2)
}
