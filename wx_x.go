package tok

import (
	"fmt"
	"time"

	"golang.org/x/net/websocket"
)

// xWsAdapter is an adapter for golang.org/x/net/websocket connections.
// It implements the conAdapter interface and provides unified read/write/timeout management for websockets.
type xWsAdapter struct {
	conn         *websocket.Conn // Underlying x websocket connection
	txt          bool            // If true, use text frames; otherwise, use binary frames
	writeTimeout time.Duration   // Timeout for write operations
	readTimeout  time.Duration   // Timeout for read operations
}

func (p *xWsAdapter) Read() ([]byte, error) {
	if p.readTimeout > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			return nil, fmt.Errorf("setting x ws read deadline err: %w", err)
		}
	}

	if p.txt {
		var s string
		err := websocket.Message.Receive(p.conn, &s)
		return []byte(s), err
	}

	var b []byte
	err := websocket.Message.Receive(p.conn, &b)
	return b, err

}

func (p *xWsAdapter) Write(b []byte) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("setting x ws write deadline failed: %w", err)
	}

	if p.txt {
		return websocket.Message.Send(p.conn, string(b))
	}

	return websocket.Message.Send(p.conn, b)
}

func (p *xWsAdapter) Close() error {
	return p.conn.Close()
}

func (p *xWsAdapter) ShareConn(adapter conAdapter) bool {
	wsAdapter, ok := adapter.(*xWsAdapter)
	if !ok {
		return false
	}
	return p.conn == wsAdapter.conn
}
