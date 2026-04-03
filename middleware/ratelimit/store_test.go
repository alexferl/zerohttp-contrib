package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/middleware/ratelimit"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/redis/go-redis/v9"
)

// mockRedisClient implements a minimal redis.Client mock for testing
type mockRedisClient struct {
	zremrangebyscore func(ctx context.Context, key string, min, max string) *redis.IntCmd
	zcard            func(ctx context.Context, key string) *redis.IntCmd
	zrange           func(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd
	zrangewithscores func(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd
	zadd             func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd
	expire           func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	incr             func(ctx context.Context, key string) *redis.IntCmd
	expirenx         func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd
	eval             func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd
}

func (m *mockRedisClient) ZRemRangeByScore(ctx context.Context, key string, min, max string) *redis.IntCmd {
	if m.zremrangebyscore != nil {
		return m.zremrangebyscore(ctx, key, min, max)
	}
	cmd := redis.NewIntCmd(ctx, "zremrangebyscore", key, min, max)
	cmd.SetVal(0)
	return cmd
}

func (m *mockRedisClient) ZCard(ctx context.Context, key string) *redis.IntCmd {
	if m.zcard != nil {
		return m.zcard(ctx, key)
	}
	cmd := redis.NewIntCmd(ctx, "zcard", key)
	cmd.SetVal(0)
	return cmd
}

func (m *mockRedisClient) ZRange(ctx context.Context, key string, start, stop int64) *redis.StringSliceCmd {
	if m.zrange != nil {
		return m.zrange(ctx, key, start, stop)
	}
	cmd := redis.NewStringSliceCmd(ctx, "zrange", key, start, stop)
	cmd.SetVal([]string{})
	return cmd
}

func (m *mockRedisClient) ZRangeWithScores(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
	if m.zrangewithscores != nil {
		return m.zrangewithscores(ctx, key, start, stop)
	}
	cmd := redis.NewZSliceCmd(ctx, "zrange", key, start, stop, "withscores")
	cmd.SetVal([]redis.Z{})
	return cmd
}

func (m *mockRedisClient) ZAdd(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
	if m.zadd != nil {
		return m.zadd(ctx, key, members...)
	}
	cmd := redis.NewIntCmd(ctx, "zadd", key)
	cmd.SetVal(1)
	return cmd
}

func (m *mockRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if m.expire != nil {
		return m.expire(ctx, key, expiration)
	}
	cmd := redis.NewBoolCmd(ctx, "expire", key, expiration)
	cmd.SetVal(true)
	return cmd
}

func (m *mockRedisClient) Incr(ctx context.Context, key string) *redis.IntCmd {
	if m.incr != nil {
		return m.incr(ctx, key)
	}
	cmd := redis.NewIntCmd(ctx, "incr", key)
	cmd.SetVal(1)
	return cmd
}

func (m *mockRedisClient) ExpireNX(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
	if m.expirenx != nil {
		return m.expirenx(ctx, key, expiration)
	}
	cmd := redis.NewBoolCmd(ctx, "expirenx", key, expiration)
	cmd.SetVal(true)
	return cmd
}

func (m *mockRedisClient) Eval(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
	if m.eval != nil {
		return m.eval(ctx, script, keys, args...)
	}
	// Default: return allowed (1), remaining, reset_time for token bucket
	cmd := redis.NewCmd(ctx, "eval", script, len(keys))
	cmd.SetVal([]interface{}{int64(1), int64(9), time.Now().Add(time.Minute).UnixMilli()})
	return cmd
}

func (m *mockRedisClient) Close() error {
	return nil
}

func TestNewRedisStore(t *testing.T) {
	client := &mockRedisClient{}
	window := time.Minute
	rate := 100

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    window,
		Rate:      rate,
	})

	zhtest.AssertNotNil(t, store)
	zhtest.AssertEqual(t, client, store.client)
	zhtest.AssertEqual(t, window, store.window)
	zhtest.AssertEqual(t, rate, store.rate)
	zhtest.AssertEqual(t, ratelimit.SlidingWindow, store.algorithm)
	zhtest.AssertEqual(t, "ratelimit:", store.keyPrefix)
}

func TestNewRedisStore_Defaults(t *testing.T) {
	client := &mockRedisClient{}

	store := NewRedisStore(client)

	zhtest.AssertNotNil(t, store)
	zhtest.AssertEqual(t, client, store.client)
	zhtest.AssertEqual(t, time.Minute, store.window)
	zhtest.AssertEqual(t, 100, store.rate)
	zhtest.AssertEqual(t, ratelimit.TokenBucket, store.algorithm)
	zhtest.AssertEqual(t, "ratelimit:", store.keyPrefix)
}

func TestRedisStore_CheckAndRecord_Allowed(t *testing.T) {
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetVal(1)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining)

	expectedReset := now.Add(time.Minute)
	timeDiff := resetTime.Sub(expectedReset)
	if timeDiff > time.Second || timeDiff < -time.Second {
		t.Errorf("expected reset time around %v, got %v", expectedReset, resetTime)
	}
}

func TestRedisStore_CheckAndRecord_Denied(t *testing.T) {
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(10) // At rate limit
			return cmd
		},
		zrangewithscores: func(ctx context.Context, key string, start, stop int64) *redis.ZSliceCmd {
			cmd := redis.NewZSliceCmd(ctx, "zrange", key, start, stop, "withscores")
			cmd.SetVal([]redis.Z{
				{Score: float64(time.Now().Add(-30 * time.Second).UnixMilli())},
			})
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertFalse(t, allowed)
	zhtest.AssertEqual(t, 0, remaining)
	zhtest.AssertTrue(t, resetTime.After(now))
}

func TestRedisStore_CheckAndRecord_RedisError(t *testing.T) {
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetErr(errors.New("redis connection error"))
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	// Should fail open (allow request) on Redis error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining) // rate - 1
}

func TestRedisStore_CheckAndRecord_ZAddError(t *testing.T) {
	callCount := 0
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			callCount++
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetErr(errors.New("zadd error"))
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	// Should fail open on ZAdd error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining)
	zhtest.AssertTrue(t, callCount > 0)
}

func TestRedisStore_KeyPrefix(t *testing.T) {
	var capturedKey string
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			capturedKey = key
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetVal(1)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	store.CheckAndRecord(context.Background(), "user-123", now)

	expectedKey := "ratelimit:sw:user-123"
	zhtest.AssertEqual(t, expectedKey, capturedKey)
}

func TestRedisStore_WindowCleanup(t *testing.T) {
	zremCalled := false
	client := &mockRedisClient{
		zremrangebyscore: func(ctx context.Context, key string, min, max string) *redis.IntCmd {
			zremCalled = true
			// Verify it's removing old entries
			zhtest.AssertEqual(t, "0", min)
			cmd := redis.NewIntCmd(ctx, "zremrangebyscore", key, min, max)
			cmd.SetVal(0)
			return cmd
		},
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetVal(1)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, zremCalled)
}

func TestRedisStore_Expire(t *testing.T) {
	expireCalled := false
	var capturedExpiration time.Duration
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetVal(1)
			return cmd
		},
		expire: func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
			expireCalled = true
			capturedExpiration = expiration
			cmd := redis.NewBoolCmd(ctx, "expire", key, expiration)
			cmd.SetVal(true)
			return cmd
		},
	}

	window := time.Minute
	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    window,
		Rate:      10,
	})
	now := time.Now()

	store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, expireCalled)
	zhtest.AssertEqual(t, window, capturedExpiration)
}

func TestRedisStore_ImplementsInterface(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      10,
	})

	// Verify RedisStore implements ratelimit.Store
	var _ ratelimit.Store = store
}

func TestRedisStore_Close(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      100,
	})

	err := store.Close()
	zhtest.AssertNoError(t, err)
}

func BenchmarkRedisStore_CheckAndRecord(b *testing.B) {
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetVal(0)
			return cmd
		},
		zadd: func(ctx context.Context, key string, members ...redis.Z) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zadd", key)
			cmd.SetVal(1)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.SlidingWindow,
		Window:    time.Minute,
		Rate:      1000,
	})
	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for b.Loop() {
		store.CheckAndRecord(ctx, fmt.Sprintf("key-%d", b.N%100), now)
	}
}

// Token Bucket Tests

func TestRedisStore_TokenBucket_Allowed(t *testing.T) {
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetVal([]interface{}{int64(1), int64(5), time.Now().Add(time.Second).UnixMilli()})
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 5, remaining)
	zhtest.AssertTrue(t, resetTime.After(now))
}

func TestRedisStore_TokenBucket_Denied(t *testing.T) {
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetVal([]interface{}{int64(0), int64(0), time.Now().Add(time.Second).UnixMilli()})
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertFalse(t, allowed)
	zhtest.AssertEqual(t, 0, remaining)
	zhtest.AssertTrue(t, resetTime.After(now))
}

func TestRedisStore_TokenBucket_RedisError(t *testing.T) {
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetErr(errors.New("redis error"))
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	// Should fail open on Redis error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining) // rate - 1
}

func TestRedisStore_TokenBucket_InvalidResult(t *testing.T) {
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetVal("invalid")
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	// Should fail open on invalid result
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining)
}

func TestRedisStore_TokenBucket_KeyPrefix(t *testing.T) {
	var capturedKey string
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			capturedKey = keys[0]
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetVal([]interface{}{int64(1), int64(9), time.Now().Add(time.Second).UnixMilli()})
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.TokenBucket,
		Window:    time.Minute,
		Rate:      10,
	})
	store.CheckAndRecord(context.Background(), "user-123", time.Now())

	expectedKey := "ratelimit:tb:user-123"
	zhtest.AssertEqual(t, expectedKey, capturedKey)
}

// Fixed Window Tests

func TestRedisStore_FixedWindow_Allowed(t *testing.T) {
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetVal(5)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 5, remaining) // rate - count
	zhtest.AssertTrue(t, resetTime.After(now))
}

func TestRedisStore_FixedWindow_Denied(t *testing.T) {
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetVal(11) // Over the limit
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertFalse(t, allowed)
	zhtest.AssertEqual(t, 0, remaining)
	zhtest.AssertTrue(t, resetTime.After(now))
}

func TestRedisStore_FixedWindow_RedisError(t *testing.T) {
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetErr(errors.New("redis error"))
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	now := time.Now()

	// Should fail open on Redis error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 9, remaining) // rate - 1
}

func TestRedisStore_FixedWindow_ExpireNXCalled(t *testing.T) {
	expireNXCalled := false
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetVal(1) // First request in window
			return cmd
		},
		expirenx: func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
			expireNXCalled = true
			cmd := redis.NewBoolCmd(ctx, "expirenx", key, expiration)
			cmd.SetVal(true)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	store.CheckAndRecord(context.Background(), "test-key", time.Now())

	zhtest.AssertTrue(t, expireNXCalled)
}

func TestRedisStore_FixedWindow_ExpireNXNotCalledOnExisting(t *testing.T) {
	expireNXCalled := false
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetVal(5) // Not the first request
			return cmd
		},
		expirenx: func(ctx context.Context, key string, expiration time.Duration) *redis.BoolCmd {
			expireNXCalled = true
			cmd := redis.NewBoolCmd(ctx, "expirenx", key, expiration)
			cmd.SetVal(true)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	store.CheckAndRecord(context.Background(), "test-key", time.Now())

	zhtest.AssertFalse(t, expireNXCalled)
}

func TestRedisStore_FixedWindow_KeyPrefix(t *testing.T) {
	var capturedKey string
	client := &mockRedisClient{
		incr: func(ctx context.Context, key string) *redis.IntCmd {
			capturedKey = key
			cmd := redis.NewIntCmd(ctx, "incr", key)
			cmd.SetVal(1)
			return cmd
		},
	}

	store := NewRedisStore(client, RedisStoreConfig{
		Algorithm: ratelimit.FixedWindow,
		Window:    time.Minute,
		Rate:      10,
	})
	store.CheckAndRecord(context.Background(), "user-123", time.Now())

	expectedKey := "ratelimit:fw:user-123"
	zhtest.AssertEqual(t, expectedKey, capturedKey)
}

// Algorithm selection test

func TestRedisStore_DefaultAlgorithm(t *testing.T) {
	client := &mockRedisClient{
		eval: func(ctx context.Context, script string, keys []string, args ...interface{}) *redis.Cmd {
			// Token bucket uses Eval
			cmd := redis.NewCmd(ctx, "eval", script, len(keys))
			cmd.SetVal([]interface{}{int64(1), int64(5), time.Now().Add(time.Second).UnixMilli()})
			return cmd
		},
	}

	// Use empty algorithm (should default to token bucket)
	store := NewRedisStore(client, RedisStoreConfig{
		Window: time.Minute,
		Rate:   10,
	})
	now := time.Now()

	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	zhtest.AssertTrue(t, allowed)
	zhtest.AssertEqual(t, 5, remaining)
}
