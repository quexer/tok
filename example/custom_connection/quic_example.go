//go:build ignore
// +build ignore

// This file demonstrates how to implement a QUIC adapter for tok.
// To use this example, you need to install the QUIC library:
//   go get github.com/quic-go/quic-go

package main

import (
	"context"
	"crypto/tls"
	"encoding/binary"
	"fmt"
	"io"
	"log"

	"github.com/quexer/tok"
	// "github.com/quic-go/quic-go"
)

// QUICAdapter implements tok.ConAdapter for QUIC streams
// Uncomment when using with quic-go library
/*
type QUICAdapter struct {
	stream quic.Stream
	conn   quic.Connection
}
*/

// For demonstration purposes without quic-go dependency
type QUICAdapter struct {
	// stream quic.Stream
	// conn   quic.Connection
	stream interface{}
	conn   interface{}
}

// Read implements tok.ConAdapter
func (q *QUICAdapter) Read() ([]byte, error) {
	// Read message length (4 bytes)
	header := make([]byte, 4)
	if _, err := io.ReadFull(q.stream, header); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header)

	// Read message body
	data := make([]byte, length)
	if _, err := io.ReadFull(q.stream, data); err != nil {
		return nil, err
	}

	return data, nil
}

// Write implements tok.ConAdapter
func (q *QUICAdapter) Write(data []byte) error {
	// Write message length
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	if _, err := q.stream.Write(header); err != nil {
		return err
	}

	// Write message body
	_, err := q.stream.Write(data)
	return err
}

// Close implements tok.ConAdapter
func (q *QUICAdapter) Close() error {
	return q.stream.Close()
}

// ShareConn implements tok.ConAdapter
func (q *QUICAdapter) ShareConn(adapter tok.ConAdapter) bool {
	other, ok := adapter.(*QUICAdapter)
	if !ok {
		return false
	}
	// Check if they share the same QUIC connection
	return q.conn == other.conn
}

// StartQUICServer demonstrates how to use QUIC with tok
// Uncomment and modify when using with quic-go library
/*
func StartQUICServer(hub *tok.Hub, addr string, tlsConfig *tls.Config) error {
	listener, err := quic.ListenAddr(addr, tlsConfig, nil)
	if err != nil {
		return err
	}

	fmt.Println("QUIC server listening on", addr)

	for {
		conn, err := listener.Accept(context.Background())
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		go handleQUICConnection(hub, conn)
	}
}

func handleQUICConnection(hub *tok.Hub, conn quic.Connection) {
	// Accept stream for this connection
	stream, err := conn.AcceptStream(context.Background())
	if err != nil {
		log.Println("Accept stream error:", err)
		return
	}

	adapter := &QUICAdapter{
		stream: stream,
		conn:   conn,
	}

	// Read authentication data
	authData, err := adapter.Read()
	if err != nil {
		log.Println("Auth read error:", err)
		adapter.Close()
		return
	}

	// Parse authentication and create device
	device := authenticate(authData)
	if device == nil {
		log.Println("Authentication failed")
		adapter.Close()
		return
	}

	// Register the connection with hub
	hub.RegisterConnection(device, adapter)

	fmt.Printf("User %s connected via QUIC\n", device.UID())
}
*/

func authenticate(data []byte) *tok.Device {
	// Simple authentication logic
	// In real applications, you would verify tokens, credentials, etc.
	userID := string(data)
	if userID == "" {
		return nil
	}

	return tok.CreateDevice(userID, "quic-session")
}

// Example: Custom adapter with compression
type CompressedAdapter struct {
	underlying tok.ConAdapter
	// Add compression logic here
}

func (c *CompressedAdapter) Read() ([]byte, error) {
	data, err := c.underlying.Read()
	if err != nil {
		return nil, err
	}
	// Decompress data here
	return data, nil
}

func (c *CompressedAdapter) Write(data []byte) error {
	// Compress data here
	return c.underlying.Write(data)
}

func (c *CompressedAdapter) Close() error {
	return c.underlying.Close()
}

func (c *CompressedAdapter) ShareConn(adapter tok.ConAdapter) bool {
	other, ok := adapter.(*CompressedAdapter)
	if !ok {
		return false
	}
	return c.underlying.ShareConn(other.underlying)
}
