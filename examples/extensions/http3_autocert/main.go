package main

import (
	"flag"
	"log"
	"net/http"

	zh "github.com/alexferl/zerohttp"
	zcautocert "github.com/alexferl/zerohttp-contrib/extensions/autocert"
	"github.com/alexferl/zerohttp-contrib/extensions/http3"
	"github.com/alexferl/zerohttp/httpx"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	domain := flag.String("domain", "", "Domain name for Let's Encrypt certificate (required)")
	flag.Parse()

	if *domain == "" {
		log.Fatal("Please provide a domain name with -domain flag")
	}

	// Create autocert manager for automatic certificates
	mgr := zcautocert.New(
		autocert.DirCache("/var/cache/certs"),
		[]string{*domain},
	)

	// Create zerohttp server with autocert manager
	app := zh.New(
		zh.Config{
			Addr: ":80",
			TLS: zh.TLSConfig{
				Addr: ":443",
			},
			Extensions: zh.ExtensionsConfig{
				AutocertManager: mgr,
			},
		},
	)

	// Add Alt-Svc header to advertise HTTP/3 support
	app.Use(func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Add(httpx.HeaderAltSvc, `h3=":443"; ma=86400`)
			next.ServeHTTP(w, r)
		})
	})

	// Add routes
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Set(httpx.HeaderContentType, httpx.MIMETextPlain)
		_, err := w.Write([]byte("Hello over HTTP/3!\n"))
		return err
	}))

	// Create HTTP/3 server with autocert support
	h3Server := http3.NewWithAutocert(":443", app, mgr)
	app.SetHTTP3Server(h3Server)

	// Start server with AutoTLS (HTTP, HTTPS, and HTTP/3)
	// This starts:
	// - HTTP server on :80 (for ACME challenges and redirects)
	// - HTTPS server on :443 (HTTP/1 and HTTP/2 with AutoTLS)
	// - HTTP/3 server on :443 (if HTTP3Server implements HTTP3ServerWithAutocert)
	log.Fatal(app.StartAutoTLS())
}
