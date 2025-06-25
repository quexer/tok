package main

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand/v2"
	"time"

	"golang.org/x/net/websocket"
)

const (
	origin = "http://localhost/"
	url    = "ws://localhost:8090/ws" // WebSocket server address
)

func main() {
	// Create WebSocket config
	config, err := websocket.NewConfig(url, origin)
	if err != nil {
		log.Fatal("fail to create websocket config", err)
	}

	config.Header.Set("Authorization", "u1")

	// Connect to WebSocket server
	conn, err := websocket.DialConfig(config)
	if err != nil {
		log.Fatal("connection fail", err)
	}
	defer conn.Close() // Close connection when function ends

	slog.Info("connected to websocket server")

	// send random message to server every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			<-ticker.C
			if err := websocket.Message.Send(conn, fmt.Sprintf("hello server %d", rand.IntN(10))); err != nil {
				log.Fatal(err)
			}
		}

	}()

	// receive messages from server
	for {
		var message string
		if err := websocket.Message.Receive(conn, &message); err != nil {
			slog.Warn("fail to receive", "err", err)
			break // quit loop
		}

		slog.Info("message received", "message", message)

		if message == "ping" {
			if err := websocket.Message.Send(conn, fmt.Sprintf("pong %d", rand.IntN(10))); err != nil {
				log.Fatal(err)
			}
		}

	}

	fmt.Println("client disconnected")
}
