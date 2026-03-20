package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcautocert "github.com/alexferl/zerohttp-contrib/extensions/autocert"
	zcwt "github.com/alexferl/zerohttp-contrib/extensions/webtransport"
	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
	"golang.org/x/crypto/acme/autocert"
)

func main() {
	domain := flag.String("domain", "", "Domain name for Let's Encrypt certificate (required)")
	flag.Parse()

	if *domain == "" {
		log.Fatal("Please provide a domain name with -domain flag")
	}

	// Create autocert manager for Let's Encrypt
	mgr := zcautocert.New(
		autocert.DirCache("/var/cache/certs"),
		[]string{*domain},
	)

	// Create zerohttp app with autocert manager
	app := zh.New(
		config.Config{
			DisableDefaultMiddlewares: true,
			Addr:                      ":80", // HTTP port for ACME challenges
			TLS: config.TLSConfig{
				Addr: ":443", // HTTPS port
			},
			Extensions: config.ExtensionsConfig{
				AutocertManager: mgr,
			},
		},
	)

	// Create HTTP/3 server
	h3 := &http3.Server{
		Addr:    ":443",
		Handler: app,
	}

	// Create WebTransport server
	wtServer := zcwt.New(h3)

	// Set WebTransport server - zerohttp will start it automatically
	app.SetWebTransportServer(wtServer)

	// Serve the HTML client
	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add("Alt-Svc", `h3=":443"; ma=86400`)
		return zh.R.File(w, r, "static/index.html")
	}))

	// WebTransport endpoint - register CONNECT handler
	app.CONNECT("/webtransport", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return err
		}
		go handleSession(sess)
		return nil
	}))

	// Start with AutoTLS - WebTransport starts automatically with Let's Encrypt!
	log.Fatal(app.StartAutoTLS())
}

func handleSession(sess *webtransport.Session) {
	defer sess.CloseWithError(0, "done")

	log.Printf("WebTransport session from %s", sess.RemoteAddr())

	// Handle datagrams
	go func() {
		for {
			msg, err := sess.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			sess.SendDatagram(append([]byte("Echo: "), msg...))
		}
	}()

	// Handle streams
	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go func(str *webtransport.Stream) {
			defer str.Close()
			buf := make([]byte, 1024)
			for {
				n, err := str.Read(buf)
				if n > 0 {
					msg := string(buf[:n])
					response := fmt.Sprintf("[%s] Echo: %s", time.Now().Format("15:04:05"), msg)
					str.Write([]byte(response))
				}
				if err != nil {
					return
				}
			}
		}(stream)
	}
}
