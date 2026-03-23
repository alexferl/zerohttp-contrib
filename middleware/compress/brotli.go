package compress

import (
	"io"

	"github.com/alexferl/zerohttp/middleware/compress"
	"github.com/andybalholm/brotli"
)

// BrotliEncoder implements compress.Encoder for Brotli compression.
// Brotli typically provides 20-26% better compression than gzip.
type BrotliEncoder struct {
	level int
}

// Encode wraps the provided io.Writer with Brotli compression.
// Uses the encoder's configured level.
func (e BrotliEncoder) Encode(w io.Writer, _ int) io.Writer {
	return brotli.NewWriterLevel(w, e.level)
}

// Encoding returns the Content-Encoding header value for Brotli.
func (e BrotliEncoder) Encoding() string {
	return "br"
}

// BrotliProvider implements compress.Provider for Brotli.
// Use this with middleware.Compress to enable Brotli compression:
//
//	app.Use(middleware.Compress(compress.Config{
//	    Algorithms: []compress.Algorithm{"br", compress.Gzip},
//	    Providers:  []compress.Provider{compress.BrotliProvider{}},
//	}))
//
// To specify a custom compression level:
//
//	app.Use(middleware.Compress(compress.Config{
//	    Algorithms: []compress.Algorithm{"br"},
//	    Providers: []compress.Provider{
//	        compress.BrotliProvider{Level: 6},
//	    },
//	}))
type BrotliProvider struct {
	// Level is the Brotli compression level to use.
	// If zero, uses level 4 (brotli.DefaultCompression).
	// Valid values: 0 (brotli.BestSpeed) to 11 (brotli.BestCompression).
	Level int
}

// GetEncoder returns a BrotliEncoder for "br" encoding.
func (p BrotliProvider) GetEncoder(encoding string) compress.Encoder {
	if encoding == "br" {
		level := p.Level
		if level == 0 {
			level = 4
		}
		return BrotliEncoder{level: level}
	}
	return nil
}
