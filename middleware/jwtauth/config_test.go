package jwtauth

import (
	"testing"

	"github.com/lestrrat-go/jwx/v3/jwa"
	"github.com/lestrrat-go/jwx/v3/jwk"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, jwa.HS256(), cfg.Algorithm)
	assert.True(t, cfg.ValidateExpiration)
	assert.True(t, cfg.ValidateNotBefore)
	assert.NotNil(t, cfg.TokenKeyFunc)
	assert.NotNil(t, cfg.KeySelector)
	assert.Nil(t, cfg.KeySet)
	assert.Nil(t, cfg.Store)
}

func TestDefaultTokenKeyFunc(t *testing.T) {
	tests := []struct {
		name     string
		claims   map[string]any
		expected string
	}{
		{
			name:     "empty claims returns colon",
			claims:   map[string]any{},
			expected: ":0",
		},
		{
			name:     "only sub claim",
			claims:   map[string]any{"sub": "user123"},
			expected: "user123:0",
		},
		{
			name:     "jti takes precedence over sid and exp",
			claims:   map[string]any{"sub": "user123", "jti": "abc123", "sid": "session456", "exp": int64(1234567890)},
			expected: "user123:abc123",
		},
		{
			name:     "sid used when jti missing",
			claims:   map[string]any{"sub": "user123", "sid": "session456", "exp": int64(1234567890)},
			expected: "user123:session456",
		},
		{
			name:     "exp used when jti and sid missing",
			claims:   map[string]any{"sub": "user123", "exp": int64(1234567890)},
			expected: "user123:1234567890",
		},
		{
			name:     "exp as float64",
			claims:   map[string]any{"sub": "user123", "exp": float64(1234567890)},
			expected: "user123:1234567890",
		},
		{
			name:     "empty jti falls back to sid",
			claims:   map[string]any{"sub": "user123", "jti": "", "sid": "session456"},
			expected: "user123:session456",
		},
		{
			name:     "empty sid falls back to exp",
			claims:   map[string]any{"sub": "user123", "sid": "", "exp": int64(1234567890)},
			expected: "user123:1234567890",
		},
		{
			name:     "both jti and sid empty falls back to exp",
			claims:   map[string]any{"sub": "user123", "jti": "", "sid": "", "exp": int64(1234567890)},
			expected: "user123:1234567890",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := defaultTokenKeyFunc(tt.claims)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDefaultKeySelector(t *testing.T) {
	t.Run("returns first key from set", func(t *testing.T) {
		rawKey := []byte("test-key-at-least-32-bytes-long!")
		key, err := jwk.Import(rawKey)
		require.NoError(t, err)

		keySet := jwk.NewSet()
		err = keySet.AddKey(key)
		require.NoError(t, err)

		result, err := defaultKeySelector(keySet, nil)
		assert.NoError(t, err)
		assert.Equal(t, key, result)
	})

	t.Run("returns error for empty key set", func(t *testing.T) {
		emptySet := jwk.NewSet()

		result, err := defaultKeySelector(emptySet, nil)
		assert.Error(t, err)
		assert.Equal(t, errNoKeys, err)
		assert.Nil(t, result)
	})

	t.Run("returns first key when multiple keys exist", func(t *testing.T) {
		rawKey1 := []byte("first-key-at-least-32-bytes-long!")
		key1, err := jwk.Import(rawKey1)
		require.NoError(t, err)
		err = key1.Set(jwk.KeyIDKey, "key1")
		require.NoError(t, err)

		rawKey2 := []byte("second-key-at-least-32-bytes-long")
		key2, err := jwk.Import(rawKey2)
		require.NoError(t, err)
		err = key2.Set(jwk.KeyIDKey, "key2")
		require.NoError(t, err)

		keySet := jwk.NewSet()
		err = keySet.AddKey(key1)
		require.NoError(t, err)
		err = keySet.AddKey(key2)
		require.NoError(t, err)

		result, err := defaultKeySelector(keySet, nil)
		assert.NoError(t, err)
		assert.Equal(t, key1, result)
	})
}
