package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestZstdEncoder_Encode(t *testing.T) {
	encoder := ZstdEncoder{}

	t.Run("encode with level 1 (fastest)", func(t *testing.T) {
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 1)
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

	t.Run("encode with level 6 (default)", func(t *testing.T) {
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 6)
		require.NotNil(t, w)

		data := []byte("Hello, Zstd compression with default level!")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})

	t.Run("encode with level 9 (best)", func(t *testing.T) {
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 9)
		require.NotNil(t, w)

		data := []byte("Hello, Zstd compression with best compression!")
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
	provider := ZstdProvider{}

	t.Run("returns encoder for zstd", func(t *testing.T) {
		encoder := provider.GetEncoder("zstd")
		assert.NotNil(t, encoder)
		assert.Equal(t, "zstd", encoder.Encoding())
	})

	t.Run("returns nil for other encodings", func(t *testing.T) {
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
		encoder := BrotliEncoder{}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 6)
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
		encoder := ZstdEncoder{}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 6)
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
