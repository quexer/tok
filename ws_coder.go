package tok

import (
	"context"
	"fmt"
	"time"

	"github.com/coder/websocket"
)

// coderWsAdapter is an adapter for github.com/coder/websocket connections.
// It implements the conAdapter interface and provides unified read/write/timeout management for websockets.
type coderWsAdapter struct {
	conn         *websocket.Conn // Underlying coder websocket connection
	ctx          context.Context // Context for the connection
	txt          bool            // If true, use text frames; otherwise, use binary frames
	writeTimeout time.Duration   // Timeout for write operations
	readTimeout  time.Duration   // Timeout for read operations
}

func (p *coderWsAdapter) Read() ([]byte, error) {
	ctx := p.ctx
	if p.readTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(p.ctx, p.readTimeout)
		defer cancel()
	}

	messageType, data, err := p.conn.Read(ctx)
	if err != nil {
		return nil, err
	}

	// Verify message type matches our configuration
	if p.txt && messageType != websocket.MessageText {
		return nil, fmt.Errorf("expected text message, got binary")
	}
	if !p.txt && messageType != websocket.MessageBinary {
		return nil, fmt.Errorf("expected binary message, got text")
	}

	return data, nil
}

func (p *coderWsAdapter) Write(b []byte) error {
	ctx := p.ctx
	if p.writeTimeout > 0 {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(p.ctx, p.writeTimeout)
		defer cancel()
	}

	var messageType websocket.MessageType
	if p.txt {
		messageType = websocket.MessageText
	} else {
		messageType = websocket.MessageBinary
	}

	return p.conn.Write(ctx, messageType, b)
}

func (p *coderWsAdapter) Close() error {
	// Send close message with normal closure code
	return p.conn.Close(websocket.StatusNormalClosure, "")
}

func (p *coderWsAdapter) ShareConn(adapter conAdapter) bool {
	coderAdapter, ok := adapter.(*coderWsAdapter)
	if !ok {
		return false
	}

	return p.conn == coderAdapter.conn
}
