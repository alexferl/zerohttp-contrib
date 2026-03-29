// Package websocket provides WebSocket support for zerohttp.
//
// This adapter integrates gorilla/websocket with zerohttp, enabling
// WebSocket connections for real-time bidirectional communication.
//
// Features:
//   - WebSocket upgrade handling
//   - Bidirectional message support
//   - Concurrent connection handling
//   - Configurable read/write buffer sizes
//
// Installation:
//
//	go get github.com/alexferl/zerohttp-contrib/extensions/websocket
//
// See https://github.com/gorilla/websocket for more information.
package websocket
