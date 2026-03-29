package http3

import (
	"context"
	"crypto/tls"
	"net/http"
	"testing"
	"time"

	"github.com/alexferl/zerohttp/extensions/http3"
	"github.com/alexferl/zerohttp/zhtest"
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
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})

	s := New(":8443", handler)

	zhtest.AssertNotNil(t, s)
	zhtest.AssertNotNil(t, s.Server)
	zhtest.AssertEqual(t, ":8443", s.Addr)
	zhtest.AssertNotNil(t, s.Handler)
}

func TestNew_NilHandler(t *testing.T) {
	s := New(":8443", nil)

	zhtest.AssertNotNil(t, s)
	zhtest.AssertNil(t, s.Handler)
}

func TestListenAndServeTLS(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	})
	s := New(":0", handler)

	// This will fail since we don't have valid certs, but it tests the code path
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLS("cert.pem", "key.pem")
	// Expected to error since certs don't exist
	_ = err
}

func TestServer_ImplementsInterfaces(t *testing.T) {
	s := New(":8443", nil)

	// Verify Server implements http3.Server
	var _ http3.Server = s

	// Verify Server implements http3.ServerWithAutocert
	var _ http3.ServerWithAutocert = s
}

func TestShutdown(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := New(":0", handler)

	// Shutdown on non-started server should not panic
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := s.Shutdown(ctx)
	// quic-go may return an error for non-started server, that's fine
	// we just want to make sure the method is callable
	_ = err
}

func TestClose(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := New(":0", handler)

	// Close should not panic
	err := s.Close()
	_ = err
}

func TestListenAndServeTLSWithAutocert_NoExistingTLSConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := New(":0", handler)

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
	// Expected to error since we're closing immediately

	// Check that TLSConfig was set
	zhtest.AssertNotNil(t, s.TLSConfig)
	zhtest.AssertNotNil(t, s.TLSConfig.GetCertificate)
	zhtest.AssertEqual(t, 1, len(s.TLSConfig.NextProtos))
	zhtest.AssertEqual(t, "h3", s.TLSConfig.NextProtos[0])

	_ = err
}

func TestListenAndServeTLSWithAutocert_WithExistingTLSConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := New(":0", handler)

	// Pre-set a custom TLSConfig
	customCiphers := []uint16{tls.TLS_AES_256_GCM_SHA384}
	s.TLSConfig = &tls.Config{
		MinVersion:   tls.VersionTLS13,
		CipherSuites: customCiphers,
	}

	manager := &mockAutocertManager{
		hostnames: []string{"example.com"},
	}

	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLSWithAutocert(manager)

	// Verify existing config is preserved
	zhtest.AssertEqual(t, tls.VersionTLS13, s.TLSConfig.MinVersion)
	zhtest.AssertEqual(t, 1, len(s.TLSConfig.CipherSuites))
	zhtest.AssertEqual(t, tls.TLS_AES_256_GCM_SHA384, s.TLSConfig.CipherSuites[0])

	// Verify autocert settings were NOT applied (user's config takes precedence)
	zhtest.AssertNil(t, s.TLSConfig.GetCertificate)

	_ = err
}

func TestNewWithAutocert(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	manager := &mockAutocertManager{
		hostnames: []string{"example.com"},
	}

	s := NewWithAutocert(":443", handler, manager)

	zhtest.AssertNotNil(t, s)
	zhtest.AssertNotNil(t, s.TLSConfig)
	zhtest.AssertNotNil(t, s.TLSConfig.GetCertificate)
	zhtest.AssertEqual(t, 1, len(s.TLSConfig.NextProtos))
	zhtest.AssertEqual(t, "h3", s.TLSConfig.NextProtos[0])
}

func TestNewWithAutocert_PreservesExistingTLSConfig(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	manager := &mockAutocertManager{
		hostnames: []string{"example.com"},
	}

	s := NewWithAutocert(":443", handler, manager)

	// Pre-set a custom TLSConfig before calling NewWithAutocert
	// Actually, we need to set it manually after NewWithAutocert to test preservation
	s.TLSConfig = &tls.Config{
		MinVersion: tls.VersionTLS13,
	}

	// Now if we call ListenAndServeTLSWithAutocert, it should use existing config
	go func() {
		time.Sleep(50 * time.Millisecond)
		_ = s.Close()
	}()

	err := s.ListenAndServeTLSWithAutocert(manager)

	zhtest.AssertEqual(t, tls.VersionTLS13, s.TLSConfig.MinVersion)

	_ = err
}

func TestServer_EmbeddedServerAccess(t *testing.T) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {})
	s := New(":8443", handler)

	// Test that we can access embedded http3.Server fields
	zhtest.AssertEqual(t, ":8443", s.Addr)

	// Modify embedded server
	s.Addr = ":443"
	zhtest.AssertEqual(t, ":443", s.Addr)
}
