package compress

import (
	"io"

	"github.com/alexferl/zerohttp/config"
	"github.com/klauspost/compress/zstd"
)

// ZstdEncoder implements config.CompressionEncoder for Zstd compression.
// Zstd provides excellent compression ratios with very fast decompression.
type ZstdEncoder struct{}

// Encode wraps the provided io.Writer with Zstd compression.
// The level parameter is mapped from gzip's 1-9 range to Zstd's speed levels.
func (e ZstdEncoder) Encode(w io.Writer, level int) io.Writer {
	// zstd levels are 1-22 (SpeedFastest to SpeedBest)
	// Map standard 1-9 to zstd range
	var zstdLevel zstd.EncoderLevel
	switch {
	case level <= 1:
		zstdLevel = zstd.SpeedFastest
	case level <= 3:
		zstdLevel = zstd.SpeedDefault
	case level <= 6:
		zstdLevel = zstd.SpeedBetterCompression
	default:
		zstdLevel = zstd.SpeedBestCompression
	}

	encoder, err := zstd.NewWriter(w, zstd.WithEncoderLevel(zstdLevel))
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

// ZstdProvider implements config.CompressionProvider for Zstd.
// Use this with middleware.Compress to enable Zstd compression:
//
//	app.Use(middleware.Compress(config.CompressConfig{
//	    Level:      6,
//	    Algorithms: []config.CompressionAlgorithm{"zstd", config.Gzip},
//	    Providers:  []config.CompressionProvider{compress.ZstdProvider{}},
//	}))
type ZstdProvider struct{}

// GetEncoder returns a ZstdEncoder for "zstd" encoding.
func (p ZstdProvider) GetEncoder(encoding string) config.CompressionEncoder {
	if encoding == "zstd" {
		return ZstdEncoder{}
	}
	return nil
}
