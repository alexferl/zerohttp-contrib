// Package http3 provides an adapter for quic-go/http3 to work with zerohttp.
package http3

import (
	"context"
	"crypto/tls"
	"net/http"

	zautocert "github.com/alexferl/zerohttp/extensions/autocert"
	zhttp3 "github.com/alexferl/zerohttp/extensions/http3"
	"github.com/quic-go/quic-go/http3"
)

var (
	_ zhttp3.Server             = (*Server)(nil)
	_ zhttp3.ServerWithAutocert = (*Server)(nil)
)

// Server wraps quic-go's http3.Server to implement zerohttp's Server interface.
type Server struct {
	*http3.Server
}

// New creates a new HTTP/3 server adapter.
func New(addr string, handler http.Handler) *Server {
	return &Server{
		Server: &http3.Server{
			Addr:    addr,
			Handler: handler,
		},
	}
}

// ListenAndServeTLS starts the HTTP/3 server with the provided certificate and key.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the HTTP/3 server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.Server.Shutdown(ctx)
}

// Close immediately closes the HTTP/3 server.
func (s *Server) Close() error {
	return s.Server.Close()
}

// ListenAndServeTLSWithAutocert starts the HTTP/3 server with automatic
// certificate management using the provided autocert manager.
// If s.TLSConfig is already set, it will be used as-is (caller is
// responsible for configuring GetCertificate and NextProtos).
// If not set, a TLSConfig with autocert settings will be created.
func (s *Server) ListenAndServeTLSWithAutocert(manager zautocert.Manager) error {
	if s.TLSConfig == nil {
		s.TLSConfig = &tls.Config{
			GetCertificate: manager.GetCertificate,
			NextProtos:     []string{"h3"},
		}
	}
	return s.ListenAndServe()
}

// NewWithAutocert creates a new HTTP/3 server pre-configured for autocert.
// This is a convenience function for use with StartAutoTLS.
// If the server already has a TLSConfig set, it will be used as-is.
// Example:
//
//	mgr := autocert.New(autocert.DirCache("/var/cache/certs"), "example.com")
//	app := zerohttp.New(config.WithAutocertManager(mgr))
//	h3 := http3.NewWithAutocert(":443", app, mgr)
//	app.SetHTTP3Server(h3)
//	app.StartAutoTLS()
func NewWithAutocert(addr string, handler http.Handler, manager zautocert.Manager) *Server {
	s := New(addr, handler)
	if s.TLSConfig == nil {
		s.TLSConfig = &tls.Config{
			GetCertificate: manager.GetCertificate,
			NextProtos:     []string{"h3"},
		}
	}
	return s
}
