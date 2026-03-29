package idempotency

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/middleware/idempotency"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/alicebob/miniredis/v2"
	"github.com/redis/go-redis/v9"
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

	zhtest.AssertNotNil(t, store)
	zhtest.AssertEqual(t, "test", store.keyPrefix)
	zhtest.AssertEqual(t, 30*time.Second, store.lockTTL)
}

func TestNewRedisStoreWithLockTTL(t *testing.T) {
	_, client := setupTestRedis(t)
	store := NewRedisStoreWithLockTTL(client, "test", 10*time.Second)

	zhtest.AssertNotNil(t, store)
	zhtest.AssertEqual(t, "test", store.keyPrefix)
	zhtest.AssertEqual(t, 10*time.Second, store.lockTTL)
}

func TestRedisStore_SetAndGet(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	record := idempotency.Record{
		StatusCode: 200,
		Headers:    []string{"Content-Type", "application/json"},
		Body:       []byte(`{"success":true}`),
		CreatedAt:  time.Now().UTC(),
	}

	// Set the record
	err := store.Set(ctx, "key1", record, time.Hour)
	zhtest.AssertNoError(t, err)

	// Get the record
	retrieved, found, err := store.Get(ctx, "key1")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, found)
	zhtest.AssertEqual(t, record.StatusCode, retrieved.StatusCode)
	zhtest.AssertDeepEqual(t, record.Headers, retrieved.Headers)
	zhtest.AssertDeepEqual(t, record.Body, retrieved.Body)
}

func TestRedisStore_Get_NotFound(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	retrieved, found, err := store.Get(ctx, "nonexistent")
	zhtest.AssertNoError(t, err)
	zhtest.AssertFalse(t, found)
	zhtest.AssertEqual(t, 0, retrieved.StatusCode)
}

func TestRedisStore_Set_WithTTL(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	record := idempotency.Record{
		StatusCode: 201,
		Body:       []byte(`created`),
	}

	// Set with short TTL
	err := store.Set(ctx, "key2", record, time.Minute)
	zhtest.AssertNoError(t, err)

	// Should exist immediately
	_, found, err := store.Get(ctx, "key2")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, found)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Minute)

	// Should not exist after TTL
	_, found, err = store.Get(ctx, "key2")
	zhtest.AssertNoError(t, err)
	zhtest.AssertFalse(t, found)
}

func TestRedisStore_Lock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// First lock should succeed
	acquired, err := store.Lock(ctx, "resource1")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)

	// Second lock should fail (already locked)
	acquired, err = store.Lock(ctx, "resource1")
	zhtest.AssertNoError(t, err)
	zhtest.AssertFalse(t, acquired)

	// Different resource should succeed
	acquired, err = store.Lock(ctx, "resource2")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)
}

func TestRedisStore_Unlock(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// Acquire lock
	acquired, err := store.Lock(ctx, "resource")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)

	// Unlock
	err = store.Unlock(ctx, "resource")
	zhtest.AssertNoError(t, err)

	// Should be able to lock again
	acquired, err = store.Lock(ctx, "resource")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)
}

func TestRedisStore_LockTTL(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	// Create store with short lock TTL
	store := NewRedisStoreWithLockTTL(client, "idemp", time.Minute)
	ctx := context.Background()

	// Acquire lock
	acquired, err := store.Lock(ctx, "resource")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)

	// Should not be able to lock (still held)
	acquired, err = store.Lock(ctx, "resource")
	zhtest.AssertNoError(t, err)
	zhtest.AssertFalse(t, acquired)

	// Fast-forward time in miniredis
	mr.FastForward(2 * time.Minute)

	// Should be able to acquire lock now
	acquired, err = store.Lock(ctx, "resource")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired)
}

func TestRedisStore_KeyPrefix(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "myprefix")
	ctx := context.Background()

	record := idempotency.Record{
		StatusCode: 200,
		Body:       []byte(`ok`),
	}

	err := store.Set(ctx, "mykey", record, time.Hour)
	zhtest.AssertNoError(t, err)

	// Verify key exists with prefix in Redis
	exists := mr.Exists("myprefix:mykey")
	zhtest.AssertTrue(t, exists)
}

func TestRedisStore_NoPrefix(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "")
	ctx := context.Background()

	record := idempotency.Record{
		StatusCode: 200,
		Body:       []byte(`ok`),
	}

	err := store.Set(ctx, "mykey", record, time.Hour)
	zhtest.AssertNoError(t, err)

	// Verify key exists without prefix in Redis
	exists := mr.Exists("mykey")
	zhtest.AssertTrue(t, exists)
}

func TestRedisStore_Get_InvalidData(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// Set invalid JSON directly in Redis
	err := client.Set(ctx, "idemp:badkey", "not-json", time.Hour).Err()
	zhtest.AssertNoError(t, err)

	// Get should return error for invalid data
	_, found, err := store.Get(ctx, "badkey")
	zhtest.AssertError(t, err)
	zhtest.AssertFalse(t, found)
}

func TestRedisStore_ConcurrentLocks(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")
	ctx := context.Background()

	// First goroutine acquires lock
	acquired1, err := store.Lock(ctx, "concurrent")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired1)

	// Multiple concurrent lock attempts should all fail
	for i := 0; i < 5; i++ {
		acquired, err := store.Lock(ctx, "concurrent")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, acquired)
	}

	// After unlock, one should succeed
	err = store.Unlock(ctx, "concurrent")
	zhtest.AssertNoError(t, err)

	acquired2, err := store.Lock(ctx, "concurrent")
	zhtest.AssertNoError(t, err)
	zhtest.AssertTrue(t, acquired2)
}

func TestRedisStore_Close(t *testing.T) {
	mr, client := setupTestRedis(t)
	defer mr.Close()

	store := NewRedisStore(client, "idemp")

	err := store.Close()
	zhtest.AssertNoError(t, err)
}
