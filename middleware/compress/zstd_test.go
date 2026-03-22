package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZstdEncoder_Encode(t *testing.T) {
	t.Run("encode with default level", func(t *testing.T) {
		encoder := ZstdEncoder{}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		data := []byte("Hello, Zstd!")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})

	t.Run("encode with custom level", func(t *testing.T) {
		encoder := ZstdEncoder{level: 1}
		var buf bytes.Buffer
		// Second parameter is ignored, encoder uses its own level
		w := encoder.Encode(&buf, 99)
		require.NotNil(t, w)

		data := []byte("Hello, Zstd compression!")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})
}

func TestZstdEncoder_Encoding(t *testing.T) {
	encoder := ZstdEncoder{}
	assert.Equal(t, "zstd", encoder.Encoding())
}

func TestZstdProvider_GetEncoder(t *testing.T) {
	t.Run("returns encoder with default level", func(t *testing.T) {
		provider := ZstdProvider{}
		encoder := provider.GetEncoder("zstd")
		assert.NotNil(t, encoder)
		assert.Equal(t, "zstd", encoder.Encoding())

		// Test encoding works with default level
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		data := []byte("Test data")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})

	t.Run("returns encoder with custom level", func(t *testing.T) {
		provider := ZstdProvider{Level: 3}
		encoder := provider.GetEncoder("zstd")
		assert.NotNil(t, encoder)

		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		data := []byte("Test data with custom level")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})

	t.Run("returns nil for other encodings", func(t *testing.T) {
		provider := ZstdProvider{}
		encoder := provider.GetEncoder("gzip")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("br")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("deflate")
		assert.Nil(t, encoder)
	})
}

func TestCompressionComparison(t *testing.T) {
	// Test that both encoders can compress data
	data := []byte("This is test data that will be compressed using different algorithms. " +
		"The quick brown fox jumps over the lazy dog. " +
		"Pack my box with five dozen liquor jugs.")

	t.Run("brotli compression", func(t *testing.T) {
		encoder := BrotliEncoder{level: 4}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		// Brotli should compress well
		t.Logf("Original: %d bytes, Brotli: %d bytes", len(data), buf.Len())
		assert.Less(t, buf.Len(), len(data))
	})

	t.Run("zstd compression", func(t *testing.T) {
		encoder := ZstdEncoder{level: 3}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		// Zstd should compress well
		t.Logf("Original: %d bytes, Zstd: %d bytes", len(data), buf.Len())
		assert.Less(t, buf.Len(), len(data))
	})
}
