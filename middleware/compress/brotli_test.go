package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrotliEncoder_Encode(t *testing.T) {
	t.Run("encode with default level", func(t *testing.T) {
		encoder := BrotliEncoder{}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		require.NotNil(t, w)

		data := []byte("Hello, Brotli!")
		_, err := w.Write(data)
		require.NoError(t, err)

		// Brotli writer needs to be closed to flush
		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		// Verify compression happened (output should be smaller or similar)
		assert.LessOrEqual(t, buf.Len(), len(data)+20) // header overhead
	})

	t.Run("encode with custom level", func(t *testing.T) {
		encoder := BrotliEncoder{level: 6}
		var buf bytes.Buffer
		// Second parameter is ignored, encoder uses its own level
		w := encoder.Encode(&buf, 99)
		require.NotNil(t, w)

		data := []byte("Hello, Brotli compression!")
		_, err := w.Write(data)
		require.NoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			require.NoError(t, err)
		}

		assert.Greater(t, buf.Len(), 0)
	})
}

func TestBrotliEncoder_Encoding(t *testing.T) {
	encoder := BrotliEncoder{}
	assert.Equal(t, "br", encoder.Encoding())
}

func TestBrotliProvider_GetEncoder(t *testing.T) {
	t.Run("returns encoder with default level", func(t *testing.T) {
		provider := BrotliProvider{}
		encoder := provider.GetEncoder("br")
		assert.NotNil(t, encoder)
		assert.Equal(t, "br", encoder.Encoding())

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
		provider := BrotliProvider{Level: 6}
		encoder := provider.GetEncoder("br")
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
		provider := BrotliProvider{}
		encoder := provider.GetEncoder("gzip")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("zstd")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("deflate")
		assert.Nil(t, encoder)
	})
}
