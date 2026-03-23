package ratelimit

import (
	"context"
	"errors"
	"fmt"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/middleware/ratelimit"
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

func TestNewRedisStore(t *testing.T) {
	client := &mockRedisClient{}
	window := time.Minute
	rate := 100

	store := NewRedisStore(client, ratelimit.SlidingWindow, window, rate)

	if store == nil {
		t.Fatal("expected non-nil store")
	}

	if store.client != client {
		t.Error("expected client to be set")
	}

	if store.window != window {
		t.Errorf("expected window %v, got %v", window, store.window)
	}

	if store.rate != rate {
		t.Errorf("expected rate %d, got %d", rate, store.rate)
	}

	if store.algorithm != ratelimit.SlidingWindow {
		t.Errorf("expected algorithm %v, got %v", ratelimit.SlidingWindow, store.algorithm)
	}

	if store.keyPrefix != "ratelimit:" {
		t.Errorf("expected keyPrefix 'ratelimit:', got %s", store.keyPrefix)
	}
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	if !allowed {
		t.Error("expected request to be allowed")
	}

	if remaining != 9 {
		t.Errorf("expected remaining 9, got %d", remaining)
	}

	expectedReset := now.Add(time.Minute)
	if resetTime.Sub(expectedReset) > time.Second {
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	allowed, remaining, resetTime := store.CheckAndRecord(context.Background(), "test-key", now)

	if allowed {
		t.Error("expected request to be denied")
	}

	if remaining != 0 {
		t.Errorf("expected remaining 0, got %d", remaining)
	}

	if resetTime.Before(now) {
		t.Error("expected reset time in the future")
	}
}

func TestRedisStore_CheckAndRecord_RedisError(t *testing.T) {
	client := &mockRedisClient{
		zcard: func(ctx context.Context, key string) *redis.IntCmd {
			cmd := redis.NewIntCmd(ctx, "zcard", key)
			cmd.SetErr(errors.New("redis connection error"))
			return cmd
		},
	}

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	// Should fail open (allow request) on Redis error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	if !allowed {
		t.Error("expected request to be allowed on Redis error (fail open)")
	}

	if remaining != 9 { // rate - 1
		t.Errorf("expected remaining 9, got %d", remaining)
	}
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	// Should fail open on ZAdd error
	allowed, remaining, _ := store.CheckAndRecord(context.Background(), "test-key", now)

	if !allowed {
		t.Error("expected request to be allowed on ZAdd error (fail open)")
	}

	if remaining != 9 {
		t.Errorf("expected remaining 9, got %d", remaining)
	}

	if callCount == 0 {
		t.Error("expected ZAdd to be called")
	}
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	store.CheckAndRecord(context.Background(), "user-123", now)

	expectedKey := "ratelimit:user-123"
	if capturedKey != expectedKey {
		t.Errorf("expected key %s, got %s", expectedKey, capturedKey)
	}
}

func TestRedisStore_WindowCleanup(t *testing.T) {
	zremCalled := false
	client := &mockRedisClient{
		zremrangebyscore: func(ctx context.Context, key string, min, max string) *redis.IntCmd {
			zremCalled = true
			// Verify it's removing old entries
			if min != "0" {
				t.Errorf("expected min '0', got %s", min)
			}
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)
	now := time.Now()

	store.CheckAndRecord(context.Background(), "test-key", now)

	if !zremCalled {
		t.Error("expected ZRemRangeByScore to be called for window cleanup")
	}
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
	store := NewRedisStore(client, ratelimit.SlidingWindow, window, 10)
	now := time.Now()

	store.CheckAndRecord(context.Background(), "test-key", now)

	if !expireCalled {
		t.Error("expected Expire to be called")
	}

	if capturedExpiration != window {
		t.Errorf("expected expiration %v, got %v", window, capturedExpiration)
	}
}

func TestRedisStore_ImplementsInterface(t *testing.T) {
	client := &mockRedisClient{}
	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 10)

	// Verify RedisStore implements ratelimit.Store
	var _ ratelimit.Store = store
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

	store := NewRedisStore(client, ratelimit.SlidingWindow, time.Minute, 1000)
	ctx := context.Background()
	now := time.Now()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		store.CheckAndRecord(ctx, fmt.Sprintf("key-%d", i%100), now)
	}
}
