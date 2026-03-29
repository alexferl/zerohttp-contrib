package webtransport

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/extensions/webtransport"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/quic-go/quic-go/http3"
)

// mockAutocertManager implements config.AutocertManager for testing
type mockAutocertManager struct {
	getCertificate func(*tls.ClientHelloInfo) (*tls.Certificate, error)
	hostnames      []string
}

func (m *mockAutocertManager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	if m.getCertificate != nil {
		return m.getCertificate(hello)
	}
	return nil, nil
}

func (m *mockAutocertManager) HTTPHandler(fallback http.Handler) http.Handler {
	return fallback
}

func (m *mockAutocertManager) Hostnames() []string {
	return m.hostnames
}

func TestNew(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}

	s := New(h3Server)

	zhtest.AssertNotNil(t, s)
	zhtest.AssertNotNil(t, s.Server)
	zhtest.AssertEqual(t, h3Server, s.H3)
}

func TestNew_NilH3Server(t *testing.T) {
	zhtest.AssertPanic(t, func() {
		New(nil)
	})
}

func TestServer_ImplementsInterfaces(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	// Verify Server implements webtransport.Server
	var _ webtransport.Server = s

	// Verify Server implements webtransport.ServerWithAutocert
	var _ webtransport.ServerWithAutocert = s
}

func TestListenAndServeTLS(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	// This will fail since we don't have valid certs, but it tests the code path
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLS("cert.pem", "key.pem")
	// Expected to error since certs don't exist
	_ = err
}

func TestShutdown(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	// Shutdown on non-started server should not panic
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.Shutdown(ctx)
	// Expected to error for non-started server
	_ = err
}

func TestClose(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	// Close should not panic even on non-started server
	err := s.Close()
	_ = err
}

func TestListenAndServeTLSWithAutocert_NoExistingTLSConfig(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	manager := &mockAutocertManager{
		hostnames: []string{"example.com"},
	}

	// This will fail since we're not actually running a server,
	// but we can verify TLSConfig is set correctly
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLSWithAutocert(manager)

	// Check that TLSConfig was set
	zhtest.AssertNotNil(t, s.H3.TLSConfig)
	zhtest.AssertNotNil(t, s.H3.TLSConfig.GetCertificate)
	zhtest.AssertEqual(t, 1, len(s.H3.TLSConfig.NextProtos))
	zhtest.AssertEqual(t, "h3", s.H3.TLSConfig.NextProtos[0])

	_ = err
}

func TestListenAndServeTLSWithAutocert_WithExistingTLSConfig(t *testing.T) {
	// Pre-set a custom TLSConfig
	customCiphers := []uint16{tls.TLS_AES_256_GCM_SHA384}
	h3Server := &http3.Server{
		Addr:    ":0",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
		TLSConfig: &tls.Config{
			MinVersion:   tls.VersionTLS13,
			CipherSuites: customCiphers,
		},
	}
	s := New(h3Server)

	manager := &mockAutocertManager{
		hostnames: []string{"example.com"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLSWithAutocert(manager)

	// Verify existing config is preserved
	zhtest.AssertEqual(t, tls.VersionTLS13, s.H3.TLSConfig.MinVersion)
	zhtest.AssertEqual(t, 1, len(s.H3.TLSConfig.CipherSuites))
	zhtest.AssertEqual(t, tls.TLS_AES_256_GCM_SHA384, s.H3.TLSConfig.CipherSuites[0])

	// Verify autocert settings were NOT applied (user's config takes precedence)
	zhtest.AssertNil(t, s.H3.TLSConfig.GetCertificate)

	_ = err
}

func TestServer_EmbeddedServerAccess(t *testing.T) {
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {}),
	}
	s := New(h3Server)

	// Test that we can access the H3 server
	zhtest.AssertEqual(t, ":8443", s.H3.Addr)
}

func TestServer_H3ServerReuse(t *testing.T) {
	// Verify that the same h3Server instance is used
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	h3Server := &http3.Server{
		Addr:    ":8443",
		Handler: handler,
	}

	s := New(h3Server)

	// Modify the original h3Server, changes should be visible
	h3Server.Addr = ":443"
	zhtest.AssertEqual(t, ":443", s.H3.Addr)
}
