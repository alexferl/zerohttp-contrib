// Package autocert provides automatic TLS certificate management for zerohttp.
//
// This adapter integrates golang.org/x/crypto/acme/autocert with zerohttp,
// enabling automatic HTTPS certificate provisioning via Let's Encrypt.
//
// Features:
//   - Automatic certificate provisioning from Let's Encrypt
//   - HTTP-01 challenge support
//   - Certificate caching for persistence
//   - Automatic renewal before expiration
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/extensions/autocert
//
// See https://pkg.go.dev/golang.org/x/crypto/acme/autocert for more information.
package autocert
