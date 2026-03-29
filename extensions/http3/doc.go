// Package http3 provides HTTP/3 support for zerohttp.
//
// This adapter integrates quic-go/http3 with zerohttp, enabling
// HTTP/3 (QUIC) protocol support for your applications.
//
// Features:
//   - HTTP/3 server support via QUIC
//   - Automatic protocol negotiation
//   - Zero-copy response writing
//   - Compatible with standard http.Handler
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/extensions/http3
//
// See https://github.com/quic-go/quic-go for more information about QUIC and HTTP/3.
package http3
