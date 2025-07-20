package tok

import (
	"fmt"
	"time"

	"github.com/gorilla/websocket"
)

// gorillaWsAdapter is an adapter for github.com/gorilla/websocket connections.
// It implements the conAdapter interface and provides unified read/write/timeout management for websockets.
type gorillaWsAdapter struct {
	conn         *websocket.Conn // Underlying gorilla websocket connection
	txt          bool            // If true, use text frames; otherwise, use binary frames
	writeTimeout time.Duration   // Timeout for write operations
	readTimeout  time.Duration   // Timeout for read operations
}

func (p *gorillaWsAdapter) Read() ([]byte, error) {
	if p.readTimeout > 0 {
		if err := p.conn.SetReadDeadline(time.Now().Add(p.readTimeout)); err != nil {
			return nil, fmt.Errorf("setting gorilla ws read deadline err: %w", err)
		}
	}

	_, data, err := p.conn.ReadMessage()
	return data, err
}

func (p *gorillaWsAdapter) Write(b []byte) error {
	if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
		return fmt.Errorf("setting gorilla ws write deadline failed: %w", err)
	}

	var messageType int
	if p.txt {
		messageType = websocket.TextMessage
	} else {
		messageType = websocket.BinaryMessage
	}

	return p.conn.WriteMessage(messageType, b)
}

func (p *gorillaWsAdapter) Close() error {
	return p.conn.Close()
}

func (p *gorillaWsAdapter) ShareConn(adapter conAdapter) bool {
	gorillaAdapter, ok := adapter.(*gorillaWsAdapter)
	if !ok {
		return false
	}
	return p.conn == gorillaAdapter.conn
}
