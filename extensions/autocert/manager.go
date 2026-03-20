// Package autocert provides an adapter for golang.org/x/crypto/acme/autocert
// to work with zerohttp.
package autocert

import (
	"crypto/tls"
	"net/http"

	"github.com/alexferl/zerohttp/config"
	"golang.org/x/crypto/acme/autocert"
)

var _ config.AutocertManager = (*Manager)(nil)

// Manager wraps autocert.Manager to implement zerohttp's AutocertManager interface.
type Manager struct {
	*autocert.Manager
	hostnames []string
}

// New creates a new autocert manager adapter.
func New(cache autocert.Cache, hostnames []string) *Manager {
	return &Manager{
		Manager: &autocert.Manager{
			Cache:      cache,
			Prompt:     autocert.AcceptTOS,
			HostPolicy: autocert.HostWhitelist(hostnames...),
		},
		hostnames: hostnames,
	}
}

// GetCertificate returns a TLS certificate for the given client hello.
func (m *Manager) GetCertificate(hello *tls.ClientHelloInfo) (*tls.Certificate, error) {
	return m.Manager.GetCertificate(hello)
}

// HTTPHandler wraps the given handler to handle ACME HTTP-01 challenges.
func (m *Manager) HTTPHandler(fallback http.Handler) http.Handler {
	return m.Manager.HTTPHandler(fallback)
}

// Hostnames returns the list of hostnames configured for this manager.
func (m *Manager) Hostnames() []string {
	result := make([]string, len(m.hostnames))
	copy(result, m.hostnames)
	return result
}
