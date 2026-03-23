package compress

import (
	"io"

	"github.com/alexferl/zerohttp/middleware/compress"
	"github.com/klauspost/compress/zstd"
)

// ZstdEncoder implements compress.Encoder for Zstd compression.
// Zstd provides excellent compression ratios with very fast decompression.
type ZstdEncoder struct {
	level zstd.EncoderLevel
}

// Encode wraps the provided io.Writer with Zstd compression.
// Uses the encoder's configured level.
func (e ZstdEncoder) Encode(w io.Writer, _ int) io.Writer {
	encoder, err := zstd.NewWriter(w, zstd.WithEncoderLevel(e.level))
	if err != nil {
		// Fall back to default on error
		encoder, _ = zstd.NewWriter(w)
	}
	return encoder
}

// Encoding returns the Content-Encoding header value for Zstd.
func (e ZstdEncoder) Encoding() string {
	return "zstd"
}

// ZstdProvider implements compress.Provider for Zstd.
// Use this with middleware.Compress to enable Zstd compression:
//
//	app.Use(middleware.Compress(compress.Config{
//	    Algorithms: []compress.Algorithm{"zstd", compress.Gzip},
//	    Providers:  []compress.Provider{compress.ZstdProvider{}},
//	}))
//
// To specify a custom compression level:
//
//	app.Use(middleware.Compress(compress.Config{
//	    Algorithms: []compress.Algorithm{"zstd"},
//	    Providers: []compress.Provider{
//	        compress.ZstdProvider{Level: zstd.SpeedBestCompression},
//	    },
//	}))
type ZstdProvider struct {
	// Level is the zstd compression level to use.
	// If zero, uses zstd.SpeedDefault.
	// Valid values: zstd.SpeedFastest, zstd.SpeedDefault, zstd.SpeedBetterCompression, zstd.SpeedBestCompression
	// or any value from 1 (SpeedFastest) to 22 (SpeedBestCompression).
	Level zstd.EncoderLevel
}

// GetEncoder returns a ZstdEncoder for "zstd" encoding.
func (p ZstdProvider) GetEncoder(encoding string) compress.Encoder {
	if encoding == "zstd" {
		level := p.Level
		if level == 0 {
			level = zstd.SpeedDefault
		}
		return ZstdEncoder{level: level}
	}
	return nil
}
