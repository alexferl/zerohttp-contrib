package main

import (
	"log"
	"net/http"

	"github.com/alexferl/zerohttp"
	"github.com/alexferl/zerohttp-contrib/middleware/compress"
	"github.com/alexferl/zerohttp/httpx"
	zcompress "github.com/alexferl/zerohttp/middleware/compress"
)

func main() {
	app := zerohttp.New()

	// Enable compression with Brotli, Zstd, Gzip, and Deflate
	// The middleware will pick the best algorithm based on the Accept-Encoding header
	app.Use(zcompress.New(zcompress.Config{
		Level: 6,
		// Algorithms are checked in order, so put the most efficient ones first
		Algorithms: []zcompress.Algorithm{
			"br",   // Brotli - best compression
			"zstd", // Zstd - fast decompression
			zcompress.Gzip,
			zcompress.Deflate,
		},
		Providers: []zcompress.Provider{
			compress.BrotliProvider{},
			compress.ZstdProvider{},
		},
	}))

	app.GET("/", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextHTML)
		_, err := w.Write([]byte(`<!DOCTYPE html>
<html>
<head><title>Compression Demo</title></head>
<body>
<h1>Hello, Compressed World!</h1>
<p>This server supports Brotli, Zstd, Gzip, and Deflate compression.</p>
<p>The middleware automatically selects the best algorithm based on your client's Accept-Encoding header.</p>
<ul>
<li>Brotli (br): Best compression ratio, slower compression, fast decompression</li>
<li>Zstd: Excellent compression with very fast decompression</li>
<li>Gzip: Widely supported, good balance</li>
<li>Deflate: Legacy support</li>
</ul>
</body>
</html>`))
		return err
	}))

	app.GET("/api/data", zerohttp.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		return zerohttp.R.JSON(w, http.StatusOK, zerohttp.M{
			"message": "This JSON response is automatically compressed",
			"data": []string{
				"item1", "item2", "item3", "item4", "item5",
				"item6", "item7", "item8", "item9", "item10",
			},
		})
	}))

	log.Fatal(app.Start())
}
