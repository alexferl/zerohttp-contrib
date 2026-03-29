package compress

import (
	"bytes"
	"io"
	"testing"

	"github.com/alexferl/zerohttp/zhtest"
)

func TestBrotliEncoder_Encode(t *testing.T) {
	t.Run("encode with default level", func(t *testing.T) {
		encoder := BrotliEncoder{}
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		zhtest.AssertNotNil(t, w)

		data := []byte("Hello, Brotli!")
		_, err := w.Write(data)
		zhtest.AssertNoError(t, err)

		// Brotli writer needs to be closed to flush
		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			zhtest.AssertNoError(t, err)
		}

		// Verify compression happened (output should be smaller or similar)
		// Use AssertTrue with comparison since AssertLessOrEqual doesn't exist
		zhtest.AssertTrue(t, buf.Len() <= len(data)+20) // header overhead
	})

	t.Run("encode with custom level", func(t *testing.T) {
		encoder := BrotliEncoder{level: 6}
		var buf bytes.Buffer
		// Second parameter is ignored, encoder uses its own level
		w := encoder.Encode(&buf, 99)
		zhtest.AssertNotNil(t, w)

		data := []byte("Hello, Brotli compression!")
		_, err := w.Write(data)
		zhtest.AssertNoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			zhtest.AssertNoError(t, err)
		}

		zhtest.AssertGreater(t, buf.Len(), 0)
	})
}

func TestBrotliEncoder_Encoding(t *testing.T) {
	encoder := BrotliEncoder{}
	zhtest.AssertEqual(t, "br", encoder.Encoding())
}

func TestBrotliProvider_GetEncoder(t *testing.T) {
	t.Run("returns encoder with default level", func(t *testing.T) {
		provider := BrotliProvider{}
		encoder := provider.GetEncoder("br")
		zhtest.AssertNotNil(t, encoder)
		zhtest.AssertEqual(t, "br", encoder.Encoding())

		// Test encoding works with default level
		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		zhtest.AssertNotNil(t, w)

		data := []byte("Test data")
		_, err := w.Write(data)
		zhtest.AssertNoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			zhtest.AssertNoError(t, err)
		}

		zhtest.AssertGreater(t, buf.Len(), 0)
	})

	t.Run("returns encoder with custom level", func(t *testing.T) {
		provider := BrotliProvider{Level: 6}
		encoder := provider.GetEncoder("br")
		zhtest.AssertNotNil(t, encoder)

		var buf bytes.Buffer
		w := encoder.Encode(&buf, 0)
		zhtest.AssertNotNil(t, w)

		data := []byte("Test data with custom level")
		_, err := w.Write(data)
		zhtest.AssertNoError(t, err)

		if closer, ok := w.(io.Closer); ok {
			err := closer.Close()
			zhtest.AssertNoError(t, err)
		}

		zhtest.AssertGreater(t, buf.Len(), 0)
	})

	t.Run("returns nil for other encodings", func(t *testing.T) {
		provider := BrotliProvider{}
		encoder := provider.GetEncoder("gzip")
		zhtest.AssertNil(t, encoder)

		encoder = provider.GetEncoder("zstd")
		zhtest.AssertNil(t, encoder)

		encoder = provider.GetEncoder("deflate")
		zhtest.AssertNil(t, encoder)
	})
}
