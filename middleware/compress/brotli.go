package compress

import (
	"io"

	"github.com/alexferl/zerohttp/config"
	"github.com/andybalholm/brotli"
)

// BrotliEncoder implements config.CompressionEncoder for Brotli compression.
// Brotli typically provides 20-26% better compression than gzip.
type BrotliEncoder struct{}

// Encode wraps the provided io.Writer with Brotli compression.
// The level parameter is mapped from gzip's 1-9 range to Brotli's 0-11 range.
func (e BrotliEncoder) Encode(w io.Writer, level int) io.Writer {
	// brotli levels are 0-11, map gzip 1-9 to brotli range
	if level < 0 {
		level = 4
	} else if level > 11 {
		level = 11
	}
	return brotli.NewWriterLevel(w, level)
}

// Encoding returns the Content-Encoding header value for Brotli.
func (e BrotliEncoder) Encoding() string {
	return "br"
}

// BrotliProvider implements config.CompressionProvider for Brotli.
// Use this with middleware.Compress to enable Brotli compression:
//
//	app.Use(middleware.Compress(config.CompressConfig{
//	    Level:      6,
//	    Algorithms: []config.CompressionAlgorithm{"br", config.Gzip},
//	    Providers:  []config.CompressionProvider{compress.BrotliProvider{}},
//	}))
type BrotliProvider struct{}

// GetEncoder returns a BrotliEncoder for "br" encoding.
func (p BrotliProvider) GetEncoder(encoding string) config.CompressionEncoder {
	if encoding == "br" {
		return BrotliEncoder{}
	}
	return nil
}
