package main

import (
	"bufio"
	"context"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"time"

	"github.com/quexer/tok"
)

// UnixSocketAdapter implements tok.ConAdapter for Unix domain sockets
type UnixSocketAdapter struct {
	conn         net.Conn
	reader       *bufio.Reader
	readTimeout  time.Duration
	writeTimeout time.Duration
}

// Read implements tok.ConAdapter
func (u *UnixSocketAdapter) Read() ([]byte, error) {
	if u.readTimeout > 0 {
		if err := u.conn.SetReadDeadline(time.Now().Add(u.readTimeout)); err != nil {
			return nil, err
		}
	}

	// Read message length (4 bytes)
	header := make([]byte, 4)
	if _, err := io.ReadFull(u.reader, header); err != nil {
		return nil, err
	}

	length := binary.BigEndian.Uint32(header)

	// Read message body
	data := make([]byte, length)
	if _, err := io.ReadFull(u.reader, data); err != nil {
		return nil, err
	}

	return data, nil
}

// Write implements tok.ConAdapter
func (u *UnixSocketAdapter) Write(data []byte) error {
	if u.writeTimeout > 0 {
		if err := u.conn.SetWriteDeadline(time.Now().Add(u.writeTimeout)); err != nil {
			return err
		}
	}

	// Write message length
	header := make([]byte, 4)
	binary.BigEndian.PutUint32(header, uint32(len(data)))

	if _, err := u.conn.Write(header); err != nil {
		return err
	}

	// Write message body
	_, err := u.conn.Write(data)
	return err
}

// Close implements tok.ConAdapter
func (u *UnixSocketAdapter) Close() error {
	return u.conn.Close()
}

// ShareConn implements tok.ConAdapter
func (u *UnixSocketAdapter) ShareConn(adapter tok.ConAdapter) bool {
	other, ok := adapter.(*UnixSocketAdapter)
	if !ok {
		return false
	}
	return u.conn == other.conn
}

// Example actor implementation
type EchoActor struct{}

func (e *EchoActor) OnReceive(dv *tok.Device, data []byte) {
	fmt.Printf("Received from %v: %s\n", dv.UID(), string(data))
}

func main() {
	// Create hub
	config := tok.NewHubConfig(&EchoActor{})
	hub, _ := tok.CreateWsHandler(nil, tok.WithWsHandlerHubConfig(config))

	// Start Unix socket server
	socketPath := "/tmp/tok_custom.sock"

	// Clean up old socket
	_, err := net.DialUnix("unix", nil, &net.UnixAddr{Name: socketPath, Net: "unix"})
	if err != nil {
		log.Fatal("Failed to dial:", err)
	}

	listener, err := net.Listen("unix", socketPath)
	if err != nil {
		log.Fatal("Failed to listen:", err)
	}
	defer listener.Close()

	fmt.Println("Unix socket server listening on", socketPath)

	// Accept connections
	for {
		conn, err := listener.Accept()
		if err != nil {
			log.Println("Accept error:", err)
			continue
		}

		go handleUnixConnection(hub, conn)
	}
}

func handleUnixConnection(hub *tok.Hub, conn net.Conn) {
	adapter := &UnixSocketAdapter{
		conn:         conn,
		reader:       bufio.NewReader(conn),
		readTimeout:  30 * time.Second,
		writeTimeout: 10 * time.Second,
	}

	// Simple authentication: read user ID from first message
	authData, err := adapter.Read()
	if err != nil {
		log.Println("Auth read error:", err)
		adapter.Close()
		return
	}

	// Create device from auth data
	device := tok.CreateDevice(string(authData), "unix-session")

	// Register the connection with hub
	hub.RegisterConnection(context.Background(), device, adapter)

	fmt.Printf("User %s connected via Unix socket\n", device.UID())
}
