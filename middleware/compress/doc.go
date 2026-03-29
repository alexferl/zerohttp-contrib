// Package compress provides response compression middleware for zerohttp.
//
// This middleware compresses HTTP responses using Brotli and/or Zstd
// compression algorithms, reducing bandwidth usage and improving
// response times for clients that support compression.
//
// Features:
//   - Brotli compression support
//   - Zstd compression support
//   - Automatic content-type detection
//   - Configurable compression levels
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/middleware/compress
package compress
