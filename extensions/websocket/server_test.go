package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	zwebsocket "github.com/alexferl/zerohttp/extensions/websocket"
	"github.com/alexferl/zerohttp/zhtest"
	"github.com/gorilla/websocket"
)

func TestNewUpgrader(t *testing.T) {
	tests := []struct {
		name      string
		upgrader  *websocket.Upgrader
		wantCheck bool
	}{
		{
			name:      "nil upgrader uses default",
			upgrader:  nil,
			wantCheck: false,
		},
		{
			name: "custom upgrader",
			upgrader: &websocket.Upgrader{
				CheckOrigin: func(r *http.Request) bool { return true },
			},
			wantCheck: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			u := NewUpgrader(tt.upgrader)

			zhtest.AssertNotNil(t, u)
			zhtest.AssertNotNil(t, u.upgrader)
			if tt.wantCheck {
				zhtest.AssertNotNil(t, u.upgrader.CheckOrigin)
			}
		})
	}
}

func TestUpgrader_Upgrade(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}
	u := NewUpgrader(upgrader)

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := u.Upgrade(w, r)
		zhtest.AssertNoError(t, err)
		defer func() { _ = conn.Close() }()

		// Echo back
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(msgType, msg)
	}))
	defer srv.Close()

	// Connect with WebSocket client
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	zhtest.AssertNoError(t, err)
	defer func() { _ = ws.Close() }()

	// Send message
	testMsg := []byte("hello")
	err = ws.WriteMessage(websocket.TextMessage, testMsg)
	zhtest.AssertNoError(t, err)

	// Read response
	_, resp, err := ws.ReadMessage()
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, string(testMsg), string(resp))
}

func TestUpgrader_Upgrade_ImplementsInterface(t *testing.T) {
	u := NewUpgrader(nil)

	// Verify Upgrader implements config.WebSocketUpgrader
	var _ zwebsocket.Upgrader = u
}

func TestUpgrader_Upgrade_Error(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return false }, // Reject all origins
	}
	u := NewUpgrader(upgrader)

	// Create test server that expects upgrade to fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		_, err := u.Upgrade(w, r)
		zhtest.AssertError(t, err)
		// Error is expected, return 400
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	// Try to connect with WebSocket client - should fail
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	_, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	zhtest.AssertError(t, err)
}

func TestConn_ReadWrite(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gorillaConn, err := upgrader.Upgrade(w, r, nil)
		zhtest.AssertNoError(t, err)

		conn := &Conn{conn: gorillaConn}
		defer func() { _ = conn.Close() }()

		// Echo
		msgType, msg, err := conn.ReadMessage()
		if err != nil {
			return
		}
		_ = conn.WriteMessage(msgType, msg)
	}))
	defer srv.Close()

	// Connect
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	zhtest.AssertNoError(t, err)
	defer func() { _ = ws.Close() }()

	// Test message
	testMsg := []byte("test message")
	err = ws.WriteMessage(websocket.TextMessage, testMsg)
	zhtest.AssertNoError(t, err)

	_, resp, err := ws.ReadMessage()
	zhtest.AssertNoError(t, err)
	zhtest.AssertEqual(t, string(testMsg), string(resp))
}

func TestConn_RemoteAddr(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	var remoteAddr string
	var wg sync.WaitGroup
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		gorillaConn, err := upgrader.Upgrade(w, r, nil)
		zhtest.AssertNoError(t, err)

		conn := &Conn{conn: gorillaConn}
		defer func() { _ = conn.Close() }()

		remoteAddr = conn.RemoteAddr().String()
	}))
	defer srv.Close()

	// Connect
	wg.Add(1)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	zhtest.AssertNoError(t, err)
	defer func() { _ = ws.Close() }()

	wg.Wait()
	zhtest.AssertNotEmpty(t, remoteAddr)
}

func TestConn_ImplementsInterface(t *testing.T) {
	// Verify Conn implements config.WebSocketConn
	var _ zwebsocket.Connection = (*Conn)(nil)
}

func TestConn_Close(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	closeCalled := false
	var wg sync.WaitGroup
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		defer wg.Done()
		gorillaConn, err := upgrader.Upgrade(w, r, nil)
		zhtest.AssertNoError(t, err)

		conn := &Conn{conn: gorillaConn}
		err = conn.Close()
		if err == nil {
			closeCalled = true
		}
	}))
	defer srv.Close()

	// Connect and immediately close
	wg.Add(1)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	zhtest.AssertNoError(t, err)
	_ = ws.Close()

	wg.Wait()
	zhtest.AssertTrue(t, closeCalled)
}
