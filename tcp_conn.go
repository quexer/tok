/**
 * tcp connection adapter
 */

package tok

import (
	"bytes"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log/slog"
	"net"
	"time"
)

const (
	tcpHeaderLen         = 4
	TCPMaxPackLen uint32 = 4 * 1024 * 1024
)

type tcpAdapter struct {
	conn         net.Conn
	readTimeout  time.Duration
	writeTimeout time.Duration
}

func (p *tcpAdapter) Read() ([]byte, error) {
	var deadline time.Time
	if p.readTimeout > 0 {
		deadline = time.Now().Add(p.readTimeout)
	} else {
		deadline = time.Time{}
	}
	if err := p.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("setting read deadline error: %w", err)
	}

	// read header
	b := make([]byte, tcpHeaderLen)
	if _, err := io.ReadFull(p.conn, b); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	var n uint32
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	if n > TCPMaxPackLen {
		return nil, fmt.Errorf("pack length %dM can't greater than %dM", n/1024/1024, TCPMaxPackLen/1024/1024)
	}

	if p.readTimeout > 0 {
		deadline = time.Now().Add(p.readTimeout)
	} else {
		deadline = time.Time{}
	}
	if err := p.conn.SetReadDeadline(deadline); err != nil {
		return nil, fmt.Errorf("setting read deadline err: %w", err)
	}

	b = make([]byte, n)
	_, err := io.ReadFull(p.conn, b)
	return b, err

}

func (p *tcpAdapter) Write(b []byte) error {
	if p.writeTimeout > 0 {
		if err := p.conn.SetWriteDeadline(time.Now().Add(p.writeTimeout)); err != nil {
			return fmt.Errorf("setting write deadline err: %w", err)
		}
	}

	n := uint32(len(b))

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, &n); err != nil {
		return err
	}
	_, err := p.conn.Write(append(buf.Bytes(), b...))

	return err
}

func (p *tcpAdapter) Close() error {
	return p.conn.Close()
}

func (p *tcpAdapter) ShareConn(adapter ConAdapter) bool {
	tcpAdp, ok := adapter.(*tcpAdapter)
	if !ok {
		return false
	}
	return p.conn == tcpAdp.conn
}

// Listen creates a TCP listener and a Hub.
// addr is the TCP address to listen on.
// auth is the function used for user authorization.
// Returns the Hub and an error if listen failed.
func Listen(ctx context.Context, config *HubConfig, addr string, auth TCPAuthFunc) (*Hub, error) {
	hub, err := createHub(ctx, config)
	if err != nil {
		return nil, fmt.Errorf("create hub err: %w", err)
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		hub.Close()
		return nil, fmt.Errorf("listen err: %w", err)
	}

	initAuth := func(conn net.Conn) {
		slog.Debug("raw tcp connection", "addr", conn.RemoteAddr())
		if err := conn.SetReadDeadline(time.Now().Add(config.authTimeout)); err != nil {
			slog.Warn("set auth deadline err", "err", err)
			_ = conn.Close()
			return
		}

		// set auth timeout at auth stage
		adapter := &tcpAdapter{
			conn:         conn,
			readTimeout:  config.authTimeout,
			writeTimeout: config.writeTimeout,
		}
		b, err := adapter.Read()
		if err != nil {
			slog.Warn("tcp auth, read err", "err", err)
			_ = adapter.Close()
			return
		}

		dv, err := auth(b)
		if err != nil {
			slog.Warn("tcp auth, auth err", "err", err)
			_ = adapter.Close()
			return
		}

		if config.readTimeout > 0 {
			adapter.readTimeout = config.readTimeout
		} else {
			adapter.readTimeout = 0
		}

		hub.RegisterConnection(hub.ctx, dv, adapter)
	}

	// Close listener when hub shuts down so Accept returns an error and the loop exits.
	go func() {
		<-hub.ctx.Done()
		_ = listener.Close()
	}()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				// If hub context is done, exit gracefully
				select {
				case <-hub.ctx.Done():
					return
				default:
				}
				slog.Warn("Error accepting", "err", err)
				continue
			}

			go initAuth(conn)
		}
	}()

	return hub, nil
}

// TCPAuthFunc tcp auth function
// parameter is the first package content of connection. return Device interface
type TCPAuthFunc func([]byte) (*Device, error)
