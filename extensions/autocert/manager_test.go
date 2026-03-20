package autocert

import (
	"context"
	"crypto/tls"
	"net/http"
	"net/http/httptest"
	"testing"

	"golang.org/x/crypto/acme/autocert"
)

// mockCache implements autocert.Cache for testing
type mockCache struct {
	data map[string][]byte
}

func newMockCache() *mockCache {
	return &mockCache{data: make(map[string][]byte)}
}

func (m *mockCache) Get(ctx context.Context, key string) ([]byte, error) {
	if v, ok := m.data[key]; ok {
		return v, nil
	}
	return nil, autocert.ErrCacheMiss
}

func (m *mockCache) Put(ctx context.Context, key string, data []byte) error {
	m.data[key] = data
	return nil
}

func (m *mockCache) Delete(ctx context.Context, key string) error {
	delete(m.data, key)
	return nil
}

func TestNew(t *testing.T) {
	cache := newMockCache()
	hostnames := []string{"example.com", "www.example.com"}

	m := New(cache, hostnames)

	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if m.Manager == nil {
		t.Error("expected embedded Manager to be set")
	}

	if len(m.hostnames) != len(hostnames) {
		t.Errorf("expected %d hostnames, got %d", len(hostnames), len(m.hostnames))
	}

	for i, h := range hostnames {
		if m.hostnames[i] != h {
			t.Errorf("expected hostname[%d] = %s, got %s", i, h, m.hostnames[i])
		}
	}
}

func TestNew_EmptyHostnames(t *testing.T) {
	cache := newMockCache()
	m := New(cache, []string{})

	if m == nil {
		t.Fatal("expected non-nil manager")
	}

	if len(m.hostnames) != 0 {
		t.Errorf("expected 0 hostnames, got %d", len(m.hostnames))
	}
}

func TestHostnames(t *testing.T) {
	cache := newMockCache()
	hostnames := []string{"example.com", "sub.example.com"}

	m := New(cache, hostnames)
	got := m.Hostnames()

	if len(got) != len(hostnames) {
		t.Errorf("expected %d hostnames, got %d", len(hostnames), len(got))
	}

	for i, h := range hostnames {
		if got[i] != h {
			t.Errorf("expected hostname[%d] = %s, got %s", i, h, got[i])
		}
	}
}

func TestHostnames_ReturnsCopy(t *testing.T) {
	cache := newMockCache()
	hostnames := []string{"example.com"}

	m := New(cache, hostnames)
	got := m.Hostnames()

	// Modify returned slice
	got[0] = "modified.com"

	// Original should be unchanged
	if m.hostnames[0] != "example.com" {
		t.Error("Hostnames() should return a copy, not the internal slice")
	}
}

func TestHTTPHandler(t *testing.T) {
	cache := newMockCache()
	m := New(cache, []string{"example.com"})

	fallbackCalled := false
	fallback := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		fallbackCalled = true
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("fallback"))
	})

	handler := m.HTTPHandler(fallback)

	// Test that handler is returned
	if handler == nil {
		t.Fatal("expected non-nil handler")
	}

	// Test request (not an ACME challenge, should hit fallback)
	req := httptest.NewRequest(http.MethodGet, "/not-acme", nil)
	rec := httptest.NewRecorder()

	handler.ServeHTTP(rec, req)

	if !fallbackCalled {
		t.Error("expected fallback handler to be called")
	}
}

func TestGetCertificate(t *testing.T) {
	cache := newMockCache()
	m := New(cache, []string{"example.com"})

	// Test with a client hello for a non-configured hostname
	// This will fail since we don't have a real ACME setup, but it tests the delegation
	hello := &tls.ClientHelloInfo{
		ServerName: "other.com",
	}

	// Expect this to fail since the hostname is not whitelisted
	_, err := m.GetCertificate(hello)
	if err == nil {
		t.Error("expected error for non-whitelisted hostname")
	}
}

func TestManager_ImplementsInterface(t *testing.T) {
	cache := newMockCache()
	m := New(cache, []string{"example.com"})

	// Verify the manager implements config.AutocertManager
	var _ interface {
		GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error)
		HTTPHandler(fallback http.Handler) http.Handler
		Hostnames() []string
	} = m
}
