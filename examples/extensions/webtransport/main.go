package main

import (
	"context"
	"crypto/tls"
	"fmt"
	"log"
	"net/http"
	"time"

	zh "github.com/alexferl/zerohttp"
	zcwt "github.com/alexferl/zerohttp-contrib/extensions/webtransport"
	"github.com/alexferl/zerohttp/httpx"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

func main() {
	certFile, keyFile := "localhost+2.pem", "localhost+2-key.pem"

	app := zh.New(
		zh.Config{
			DisableDefaultMiddlewares: true,
			TLS: zh.TLSConfig{
				Addr:     ":8443",
				CertFile: certFile,
				KeyFile:  keyFile,
			},
		},
	)

	// Create HTTP/3 server for WebTransport
	h3 := &http3.Server{
		Addr: ":8443",
		TLSConfig: &tls.Config{
			NextProtos: []string{"h3"},
		},
		Handler: app,
	}

	// Create WebTransport server (using underlying type for full control)
	wtServer := &webtransport.Server{
		H3:          h3,
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Configure HTTP/3 server for WebTransport
	webtransport.ConfigureHTTP3Server(h3)
	app.SetWebTransportServer(zcwt.New(h3))

	app.GET("/", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		w.Header().Add(httpx.HeaderAltSvc, `h3=":8443"; ma=86400`)
		return zh.R.File(w, r, "static/index.html")
	}))

	app.CONNECT("/wt", zh.HandlerFunc(func(w http.ResponseWriter, r *http.Request) error {
		sess, err := wtServer.Upgrade(w, r)
		if err != nil {
			return err
		}
		go handleSession(sess)
		return nil
	}))

	if err := app.ListenAndServeTLS(certFile, keyFile); err != nil {
		log.Fatal(err)
	}
}

func handleSession(sess *webtransport.Session) {
	defer func() {
		if err := sess.CloseWithError(0, "done"); err != nil {
			log.Printf("Error closing session: %v", err)
		}
	}()

	log.Printf("WebTransport session from %s", sess.RemoteAddr())

	go func() {
		for {
			msg, err := sess.ReceiveDatagram(context.Background())
			if err != nil {
				return
			}
			if err := sess.SendDatagram(append([]byte("Echo: "), msg...)); err != nil {
				log.Printf("Error sending datagram: %v", err)
				return
			}
		}
	}()

	for {
		stream, err := sess.AcceptStream(context.Background())
		if err != nil {
			return
		}
		go func(str *webtransport.Stream) {
			defer func() {
				if err := str.Close(); err != nil {
					log.Printf("Error closing stream: %v", err)
				}
			}()
			buf := make([]byte, 1024)
			for {
				n, err := str.Read(buf)
				if n > 0 {
					msg := string(buf[:n])
					response := fmt.Sprintf("[%s] Echo: %s", time.Now().Format("15:04:05"), msg)
					if _, err := str.Write([]byte(response)); err != nil {
						log.Printf("Error writing to stream: %v", err)
						return
					}
				}
				if err != nil {
					return
				}
			}
		}(stream)
	}
}
