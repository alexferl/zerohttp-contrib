package storage

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/storage"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/redis/go-redis/v9"
)

// mockRedisClient is a test double for storage.Client.
type mockRedisClient struct {
	getFunc     func(ctx context.Context, key string) *redis.StringCmd
	setFunc     func(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd
	setArgsFunc func(ctx context.Context, key string, value any, args redis.SetArgs) *redis.StatusCmd
	delFunc     func(ctx context.Context, keys ...string) *redis.IntCmd
	ttlFunc     func(ctx context.Context, key string) *redis.DurationCmd
	closeFunc   func() error
}

func (m *mockRedisClient) Get(ctx context.Context, key string) *redis.StringCmd {
	if m.getFunc != nil {
		return m.getFunc(ctx, key)
	}
	return redis.NewStringCmd(context.Background(), "GET", key)
}

func (m *mockRedisClient) Set(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
	if m.setFunc != nil {
		return m.setFunc(ctx, key, value, expiration)
	}
	return redis.NewStatusCmd(context.Background(), "SET", key, value)
}

func (m *mockRedisClient) SetArgs(ctx context.Context, key string, value any, args redis.SetArgs) *redis.StatusCmd {
	if m.setArgsFunc != nil {
		return m.setArgsFunc(ctx, key, value, args)
	}
	return redis.NewStatusCmd(context.Background(), "SET", key, value)
}

func (m *mockRedisClient) Del(ctx context.Context, keys ...string) *redis.IntCmd {
	if m.delFunc != nil {
		return m.delFunc(ctx, keys...)
	}
	return redis.NewIntCmd(context.Background(), "DEL", keys)
}

func (m *mockRedisClient) TTL(ctx context.Context, key string) *redis.DurationCmd {
	if m.ttlFunc != nil {
		return m.ttlFunc(ctx, key)
	}
	return redis.NewDurationCmd(context.Background(), time.Second, "TTL", key)
}

func (m *mockRedisClient) Close() error {
	if m.closeFunc != nil {
		return m.closeFunc()
	}
	return nil
}

func TestNewRedisStorage(t *testing.T) {
	client := &mockRedisClient{}

	t.Run("with defaults", func(t *testing.T) {
		s := NewRedisStorage(client)
		zhtest.AssertNotNil(t, s)
		zhtest.AssertEqual(t, "", s.keyPrefix)
		zhtest.AssertEqual(t, 30*time.Second, s.lockTTL)
	})

	t.Run("with prefix", func(t *testing.T) {
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "test"})
		zhtest.AssertNotNil(t, s)
		zhtest.AssertEqual(t, "test", s.keyPrefix)
	})

	t.Run("with custom lock TTL", func(t *testing.T) {
		s := NewRedisStorage(client, RedisStorageConfig{LockTTL: 5 * time.Minute})
		zhtest.AssertNotNil(t, s)
		zhtest.AssertEqual(t, 5*time.Minute, s.lockTTL)
	})

	t.Run("with all options", func(t *testing.T) {
		s := NewRedisStorage(client, RedisStorageConfig{
			KeyPrefix: "myapp",
			LockTTL:   1 * time.Minute,
		})
		zhtest.AssertNotNil(t, s)
		zhtest.AssertEqual(t, "myapp", s.keyPrefix)
		zhtest.AssertEqual(t, 1*time.Minute, s.lockTTL)
	})
}

func TestRedisStorage_Get(t *testing.T) {
	ctx := context.Background()

	t.Run("key found", func(t *testing.T) {
		client := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				cmd := redis.NewStringCmd(ctx, "GET", key)
				cmd.SetVal("hello world")
				return cmd
			},
		}
		s := NewRedisStorage(client)

		val, found, err := s.Get(ctx, "mykey")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, found)
		zhtest.AssertEqual(t, "hello world", string(val))
	})

	t.Run("key not found", func(t *testing.T) {
		client := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				cmd := redis.NewStringCmd(ctx, "GET", key)
				cmd.SetErr(redis.Nil)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		val, found, err := s.Get(ctx, "missing")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, found)
		zhtest.AssertNil(t, val)
	})

	t.Run("redis error", func(t *testing.T) {
		client := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				cmd := redis.NewStringCmd(ctx, "GET", key)
				cmd.SetErr(errors.New("connection refused"))
				return cmd
			},
		}
		s := NewRedisStorage(client)

		val, found, err := s.Get(ctx, "mykey")
		zhtest.AssertError(t, err)
		zhtest.AssertFalse(t, found)
		zhtest.AssertNil(t, val)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKey string
		client := &mockRedisClient{
			getFunc: func(ctx context.Context, key string) *redis.StringCmd {
				capturedKey = key
				cmd := redis.NewStringCmd(ctx, "GET", key)
				cmd.SetErr(redis.Nil)
				return cmd
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_, _, _ = s.Get(ctx, "mykey")
		zhtest.AssertEqual(t, "myapp:mykey", capturedKey)
	})
}

func TestRedisStorage_Set(t *testing.T) {
	ctx := context.Background()

	t.Run("successful set", func(t *testing.T) {
		client := &mockRedisClient{
			setFunc: func(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
				cmd := redis.NewStatusCmd(ctx, "SET", key, value)
				cmd.SetVal("OK")
				return cmd
			},
		}
		s := NewRedisStorage(client)

		err := s.Set(ctx, "mykey", []byte("hello"), 5*time.Minute)
		zhtest.AssertNoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		client := &mockRedisClient{
			setFunc: func(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
				cmd := redis.NewStatusCmd(ctx, "SET", key, value)
				cmd.SetErr(errors.New("connection refused"))
				return cmd
			},
		}
		s := NewRedisStorage(client)

		err := s.Set(ctx, "mykey", []byte("hello"), 5*time.Minute)
		zhtest.AssertError(t, err)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKey string
		client := &mockRedisClient{
			setFunc: func(ctx context.Context, key string, value any, expiration time.Duration) *redis.StatusCmd {
				capturedKey = key
				cmd := redis.NewStatusCmd(ctx, "SET", key, value)
				return cmd
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_ = s.Set(ctx, "mykey", []byte("hello"), 5*time.Minute)
		zhtest.AssertEqual(t, "myapp:mykey", capturedKey)
	})
}

func TestRedisStorage_Delete(t *testing.T) {
	ctx := context.Background()

	t.Run("successful delete", func(t *testing.T) {
		client := &mockRedisClient{
			delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				cmd := redis.NewIntCmd(ctx, "DEL", keys)
				cmd.SetVal(1)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		err := s.Delete(ctx, "mykey")
		zhtest.AssertNoError(t, err)
	})

	t.Run("redis error", func(t *testing.T) {
		client := &mockRedisClient{
			delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				cmd := redis.NewIntCmd(ctx, "DEL", keys)
				cmd.SetErr(errors.New("connection refused"))
				return cmd
			},
		}
		s := NewRedisStorage(client)

		err := s.Delete(ctx, "mykey")
		zhtest.AssertError(t, err)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKeys []string
		client := &mockRedisClient{
			delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				capturedKeys = keys
				return redis.NewIntCmd(ctx, "DEL", keys)
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_ = s.Delete(ctx, "mykey")
		zhtest.AssertLen(t, capturedKeys, 1)
		zhtest.AssertEqual(t, "myapp:mykey", capturedKeys[0])
	})
}

func TestRedisStorage_Lock(t *testing.T) {
	ctx := context.Background()

	t.Run("lock acquired", func(t *testing.T) {
		client := &mockRedisClient{
			setArgsFunc: func(ctx context.Context, key string, value any, args redis.SetArgs) *redis.StatusCmd {
				cmd := redis.NewStatusCmd(ctx, "SET", key, value)
				cmd.SetVal("OK")
				return cmd
			},
		}
		s := NewRedisStorage(client)

		acquired, err := s.Lock(ctx, "mykey", 30*time.Second)
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, acquired)
	})

	t.Run("lock not acquired", func(t *testing.T) {
		client := &mockRedisClient{
			setArgsFunc: func(ctx context.Context, key string, value any, args redis.SetArgs) *redis.StatusCmd {
				cmd := redis.NewStatusCmd(ctx, "SET", key, value)
				cmd.SetErr(redis.Nil)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		acquired, err := s.Lock(ctx, "mykey", 30*time.Second)
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, acquired)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKey string
		client := &mockRedisClient{
			setArgsFunc: func(ctx context.Context, key string, value any, args redis.SetArgs) *redis.StatusCmd {
				capturedKey = key
				return redis.NewStatusCmd(ctx, "SET", key, value)
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_, _ = s.Lock(ctx, "mykey", 30*time.Second)
		zhtest.AssertEqual(t, "myapp:mykey:lock", capturedKey)
	})
}

func TestRedisStorage_Unlock(t *testing.T) {
	ctx := context.Background()

	t.Run("successful unlock", func(t *testing.T) {
		client := &mockRedisClient{
			delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				cmd := redis.NewIntCmd(ctx, "DEL", keys)
				cmd.SetVal(1)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		err := s.Unlock(ctx, "mykey")
		zhtest.AssertNoError(t, err)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKeys []string
		client := &mockRedisClient{
			delFunc: func(ctx context.Context, keys ...string) *redis.IntCmd {
				capturedKeys = keys
				return redis.NewIntCmd(ctx, "DEL", keys)
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_ = s.Unlock(ctx, "mykey")
		zhtest.AssertLen(t, capturedKeys, 1)
		zhtest.AssertEqual(t, "myapp:mykey:lock", capturedKeys[0])
	})
}

func TestRedisStorage_TTL(t *testing.T) {
	ctx := context.Background()

	t.Run("key with TTL", func(t *testing.T) {
		client := &mockRedisClient{
			ttlFunc: func(ctx context.Context, key string) *redis.DurationCmd {
				cmd := redis.NewDurationCmd(ctx, time.Second, "TTL", key)
				cmd.SetVal(5 * time.Minute)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		ttl, err := s.TTL(ctx, "mykey")
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, 5*time.Minute, ttl)
	})

	t.Run("key without TTL", func(t *testing.T) {
		client := &mockRedisClient{
			ttlFunc: func(ctx context.Context, key string) *redis.DurationCmd {
				cmd := redis.NewDurationCmd(ctx, time.Second, "TTL", key)
				cmd.SetVal(-1 * time.Second)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		ttl, err := s.TTL(ctx, "mykey")
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, time.Duration(0), ttl)
	})

	t.Run("key not found", func(t *testing.T) {
		client := &mockRedisClient{
			ttlFunc: func(ctx context.Context, key string) *redis.DurationCmd {
				cmd := redis.NewDurationCmd(ctx, time.Second, "TTL", key)
				cmd.SetVal(-2 * time.Second)
				return cmd
			},
		}
		s := NewRedisStorage(client)

		ttl, err := s.TTL(ctx, "mykey")
		zhtest.AssertNoError(t, err)
		zhtest.AssertEqual(t, time.Duration(0), ttl)
	})

	t.Run("with prefix", func(t *testing.T) {
		var capturedKey string
		client := &mockRedisClient{
			ttlFunc: func(ctx context.Context, key string) *redis.DurationCmd {
				capturedKey = key
				return redis.NewDurationCmd(ctx, time.Second, "TTL", key)
			},
		}
		s := NewRedisStorage(client, RedisStorageConfig{KeyPrefix: "myapp"})

		_, _ = s.TTL(ctx, "mykey")
		zhtest.AssertEqual(t, "myapp:mykey", capturedKey)
	})
}

func TestRedisStorage_Close(t *testing.T) {
	t.Run("successful close", func(t *testing.T) {
		client := &mockRedisClient{
			closeFunc: func() error {
				return nil
			},
		}
		s := NewRedisStorage(client)

		err := s.Close()
		zhtest.AssertNoError(t, err)
	})

	t.Run("close error", func(t *testing.T) {
		client := &mockRedisClient{
			closeFunc: func() error {
				return errors.New("connection error")
			},
		}
		s := NewRedisStorage(client)

		err := s.Close()
		zhtest.AssertError(t, err)
	})
}

func TestRedisStorage_InterfaceCompliance(t *testing.T) {
	client := &mockRedisClient{}
	s := NewRedisStorage(client)

	t.Run("implements Storage", func(t *testing.T) {
		var _ storage.Storage = s
	})

	t.Run("implements Locker", func(t *testing.T) {
		var _ storage.Locker = s
	})

	t.Run("implements Inspector", func(t *testing.T) {
		var _ storage.Inspector = s
	})
}

func TestRedisStorage_makeKey(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		s := NewRedisStorage(&mockRedisClient{})
		key := s.makeKey("mykey")
		zhtest.AssertEqual(t, "mykey", key)
	})

	t.Run("with prefix", func(t *testing.T) {
		s := NewRedisStorage(&mockRedisClient{}, RedisStorageConfig{KeyPrefix: "app"})
		key := s.makeKey("mykey")
		zhtest.AssertEqual(t, "app:mykey", key)
	})
}

func TestRedisStorage_makeLockKey(t *testing.T) {
	t.Run("without prefix", func(t *testing.T) {
		s := NewRedisStorage(&mockRedisClient{})
		key := s.makeLockKey("mykey")
		zhtest.AssertEqual(t, "mykey:lock", key)
	})

	t.Run("with prefix", func(t *testing.T) {
		s := NewRedisStorage(&mockRedisClient{}, RedisStorageConfig{KeyPrefix: "app"})
		key := s.makeLockKey("mykey")
		zhtest.AssertEqual(t, "app:mykey:lock", key)
	})
}
