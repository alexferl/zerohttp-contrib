package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBrotliEncoder_Encode(t *testing.T) {
	encoder := BrotliEncoder{}

	t.Run("encode with default level", func(t *testing.T) {
		var buf bytes.Buffer
		w := encoder.Encode(&buf, -1)
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

	t.Run("encode with level 6", func(t *testing.T) {
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 6)
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
	provider := BrotliProvider{}

	t.Run("returns encoder for br", func(t *testing.T) {
		encoder := provider.GetEncoder("br")
		assert.NotNil(t, encoder)
		assert.Equal(t, "br", encoder.Encoding())
	})

	t.Run("returns nil for other encodings", func(t *testing.T) {
		encoder := provider.GetEncoder("gzip")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("zstd")
		assert.Nil(t, encoder)

		encoder = provider.GetEncoder("deflate")
		assert.Nil(t, encoder)
	})
}
