package websocket

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"sync"
	"testing"

	"github.com/alexferl/zerohttp/config"
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

			if u == nil {
				t.Fatal("expected non-nil Upgrader")
			}

			if u.upgrader == nil {
				t.Error("expected internal upgrader to be set")
			}

			if tt.wantCheck && u.upgrader.CheckOrigin == nil {
				t.Error("expected CheckOrigin to be set")
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
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}
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
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = ws.Close() }()

	// Send message
	testMsg := []byte("hello")
	if err := ws.WriteMessage(websocket.TextMessage, testMsg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	// Read response
	_, resp, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if string(resp) != string(testMsg) {
		t.Errorf("expected %s, got %s", testMsg, resp)
	}
}

func TestUpgrader_Upgrade_ImplementsInterface(t *testing.T) {
	u := NewUpgrader(nil)

	// Verify Upgrader implements config.WebSocketUpgrader
	var _ config.WebSocketUpgrader = u
}

func TestUpgrader_Upgrade_Error(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return false }, // Reject all origins
	}
	u := NewUpgrader(upgrader)

	// Create test server that expects upgrade to fail
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := u.Upgrade(w, r)
		if err == nil {
			t.Error("expected upgrade to fail due to rejected origin")
			_ = conn.Close()
			return
		}
		// Error is expected, return 400
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer srv.Close()

	// Try to connect with WebSocket client - should fail
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err == nil {
		_ = ws.Close()
		t.Fatal("expected dial to fail due to rejected origin")
	}
}

func TestConn_ReadWrite(t *testing.T) {
	upgrader := &websocket.Upgrader{
		CheckOrigin: func(r *http.Request) bool { return true },
	}

	// Create test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gorillaConn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}

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
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = ws.Close() }()

	// Test message
	testMsg := []byte("test message")
	if err := ws.WriteMessage(websocket.TextMessage, testMsg); err != nil {
		t.Fatalf("write failed: %v", err)
	}

	_, resp, err := ws.ReadMessage()
	if err != nil {
		t.Fatalf("read failed: %v", err)
	}

	if string(resp) != string(testMsg) {
		t.Errorf("expected %s, got %s", testMsg, resp)
	}
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
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}

		conn := &Conn{conn: gorillaConn}
		defer func() { _ = conn.Close() }()

		remoteAddr = conn.RemoteAddr().String()
	}))
	defer srv.Close()

	// Connect
	wg.Add(1)
	wsURL := "ws" + strings.TrimPrefix(srv.URL, "http")
	ws, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	defer func() { _ = ws.Close() }()

	wg.Wait()
	if remoteAddr == "" {
		t.Error("expected RemoteAddr to be set")
	}
}

func TestConn_ImplementsInterface(t *testing.T) {
	// Verify Conn implements config.WebSocketConn
	var _ config.WebSocketConn = (*Conn)(nil)
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
		if err != nil {
			t.Errorf("upgrade failed: %v", err)
			return
		}

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
	if err != nil {
		t.Fatalf("dial failed: %v", err)
	}
	_ = ws.Close()

	wg.Wait()
	if !closeCalled {
		t.Error("expected Close to succeed")
	}
}
