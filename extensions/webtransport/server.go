// Package webtransport provides an adapter for quic-go/webtransport-go to work with zerohttp.
package webtransport

import (
	"context"
	"crypto/tls"

	"github.com/alexferl/zerohttp/config"
	"github.com/quic-go/quic-go/http3"
	"github.com/quic-go/webtransport-go"
)

var (
	_ config.WebTransportServer             = (*Server)(nil)
	_ config.WebTransportServerWithAutocert = (*Server)(nil)
)

// Server wraps webtransport.Server to implement zerohttp's WebTransportServer interface.
type Server struct {
	*webtransport.Server
}

// New creates a new WebTransport server adapter.
// The h3Server parameter must not be nil. The h3Server's Handler
// should already be configured with the application handler.
func New(h3Server *http3.Server) *Server {
	if h3Server == nil {
		panic("webtransport: h3Server cannot be nil")
	}
	return &Server{
		Server: &webtransport.Server{
			H3: h3Server,
		},
	}
}

// ListenAndServeTLS starts the WebTransport server with the provided certificate and key.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	return s.Server.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the WebTransport server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.H3.Shutdown(ctx)
}

// Close immediately closes the WebTransport server.
func (s *Server) Close() error {
	return s.Server.Close()
}

// ListenAndServeTLSWithAutocert starts the WebTransport server with automatic
// certificate management using the provided autocert manager.
// If s.H3.TLSConfig is already set, it will be used as-is (caller is
// responsible for configuring GetCertificate and NextProtos).
// If not set, a TLSConfig with autocert settings will be created.
func (s *Server) ListenAndServeTLSWithAutocert(manager config.AutocertManager) error {
	if s.H3.TLSConfig == nil {
		s.H3.TLSConfig = &tls.Config{
			GetCertificate: manager.GetCertificate,
			NextProtos:     []string{"h3"},
		}
	}
	return s.ListenAndServe()
}
