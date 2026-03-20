package websocket

import (
	"net"
	"net/http"

	"github.com/alexferl/zerohttp/config"
	"github.com/gorilla/websocket"
)

// Upgrader wraps gorilla/websocket to implement config.WebSocketUpgrader
type Upgrader struct {
	upgrader *websocket.Upgrader
}

// NewUpgrader creates a new WebSocket upgrader adapter.
// If upgrader is nil, a default gorilla/websocket.Upgrader is used.
func NewUpgrader(upgrader *websocket.Upgrader) *Upgrader {
	if upgrader == nil {
		upgrader = &websocket.Upgrader{}
	}
	return &Upgrader{upgrader: upgrader}
}

func (m *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (config.WebSocketConn, error) {
	conn, err := m.upgrader.Upgrade(w, r, nil)
	if err != nil {
		return nil, err
	}
	return &Conn{conn: conn}, nil
}

// Conn wraps gorilla/websocket.Conn to implement config.WebSocketConn
type Conn struct {
	conn *websocket.Conn
}

func (c *Conn) ReadMessage() (int, []byte, error) {
	return c.conn.ReadMessage()
}

func (c *Conn) WriteMessage(messageType int, data []byte) error {
	return c.conn.WriteMessage(messageType, data)
}

func (c *Conn) Close() error {
	return c.conn.Close()
}

func (c *Conn) RemoteAddr() net.Addr {
	return c.conn.RemoteAddr()
}
