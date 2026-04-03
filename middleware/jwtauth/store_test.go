package jwtauth

import (
	"context"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/storage"
	"github.com/alexferl/zerohttp/zhtest"
)

// memoryStorage is a simple in-memory implementation of storage.Storage for testing.
type memoryStorage struct {
	data map[string][]byte
}

func newMemoryStorage() *memoryStorage {
	return &memoryStorage{data: make(map[string][]byte)}
}

func (m *memoryStorage) Get(_ context.Context, key string) ([]byte, bool, error) {
	val, ok := m.data[key]
	return val, ok, nil
}

func (m *memoryStorage) Set(_ context.Context, key string, val []byte, _ time.Duration) error {
	m.data[key] = val
	return nil
}

func (m *memoryStorage) Delete(_ context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func (m *memoryStorage) Close() error {
	return nil
}

func TestNewStorageAdapter(t *testing.T) {
	t.Run("with custom prefix", func(t *testing.T) {
		store := newMemoryStorage()
		adapter := NewStorageAdapter(store, StorageAdapterConfig{KeyPrefix: "custom:"})

		zhtest.AssertNotNil(t, adapter)
		zhtest.AssertEqual(t, "custom:", adapter.prefix)
	})

	t.Run("with defaults", func(t *testing.T) {
		store := newMemoryStorage()
		adapter := NewStorageAdapter(store)

		zhtest.AssertEqual(t, "jwt", adapter.prefix)
	})
}

func TestStorageAdapter_tokenKey(t *testing.T) {
	store := newMemoryStorage()

	tests := []struct {
		name     string
		prefix   string
		key      string
		expected string
	}{
		{
			name:     "with custom prefix",
			prefix:   "test",
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
			adapter := NewStorageAdapter(store, StorageAdapterConfig{KeyPrefix: tt.prefix})
			result := adapter.tokenKey(tt.key)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestStorageAdapter_sessionKey(t *testing.T) {
	store := newMemoryStorage()

	tests := []struct {
		name     string
		prefix   string
		sid      string
		expected string
	}{
		{
			name:     "with custom prefix",
			prefix:   "test",
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
			adapter := NewStorageAdapter(store, StorageAdapterConfig{KeyPrefix: tt.prefix})
			result := adapter.sessionKey(tt.sid)
			zhtest.AssertEqual(t, tt.expected, result)
		})
	}
}

func TestStorageAdapter(t *testing.T) {
	store := newMemoryStorage()
	adapter := NewStorageAdapter(store, StorageAdapterConfig{KeyPrefix: "test:"})
	ctx := context.Background()

	t.Run("revoke and check token", func(t *testing.T) {
		err := adapter.RevokeToken(ctx, "token-123", 15*time.Minute)
		zhtest.AssertNoError(t, err)

		revoked, err := adapter.IsTokenRevoked(ctx, "token-123")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, revoked)

		// Non-revoked token
		revoked, err = adapter.IsTokenRevoked(ctx, "token-456")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, revoked)
	})

	t.Run("revoke and check session", func(t *testing.T) {
		err := adapter.RevokeSession(ctx, "session-abc", 7*24*time.Hour)
		zhtest.AssertNoError(t, err)

		revoked, err := adapter.IsSessionRevoked(ctx, "session-abc")
		zhtest.AssertNoError(t, err)
		zhtest.AssertTrue(t, revoked)

		// Non-revoked session
		revoked, err = adapter.IsSessionRevoked(ctx, "session-def")
		zhtest.AssertNoError(t, err)
		zhtest.AssertFalse(t, revoked)
	})

	t.Run("close adapter", func(t *testing.T) {
		err := adapter.Close()
		zhtest.AssertNoError(t, err)
	})
}

func TestStorageAdapter_IsTokenRevoked_Error(t *testing.T) {
	// Test with a storage that returns an error
	errStorage := &errorStorage{err: context.Canceled}
	adapter := NewStorageAdapter(errStorage, StorageAdapterConfig{KeyPrefix: "test:"})
	ctx := context.Background()

	_, err := adapter.IsTokenRevoked(ctx, "token-123")
	zhtest.AssertError(t, err)
}

func TestStorageAdapter_IsSessionRevoked_Error(t *testing.T) {
	// Test with a storage that returns an error
	errStorage := &errorStorage{err: context.Canceled}
	adapter := NewStorageAdapter(errStorage, StorageAdapterConfig{KeyPrefix: "test:"})
	ctx := context.Background()

	_, err := adapter.IsSessionRevoked(ctx, "session-abc")
	zhtest.AssertError(t, err)
}

// errorStorage is a storage that always returns an error (for testing error handling).
type errorStorage struct {
	err error
}

func (e *errorStorage) Get(_ context.Context, _ string) ([]byte, bool, error) {
	return nil, false, e.err
}

func (e *errorStorage) Set(_ context.Context, _ string, _ []byte, _ time.Duration) error {
	return e.err
}

func (e *errorStorage) Delete(_ context.Context, _ string) error {
	return e.err
}

func (e *errorStorage) Close() error {
	return e.err
}

// Compile-time interface check
var (
	_ storage.Storage = (*memoryStorage)(nil)
	_ storage.Storage = (*errorStorage)(nil)
)
