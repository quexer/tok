package main

import (
	"fmt"
	"log"
	"log/slog"
	"math/rand/v2"
	"net/url"
	"time"

	"github.com/gorilla/websocket"
)

const (
	serverURL = "ws://localhost:8091/ws" // Gorilla WebSocket server address
)

func main() {
	// Parse the WebSocket URL
	u, err := url.Parse(serverURL)
	if err != nil {
		log.Fatal("fail to parse websocket url", err)
	}

	// Create headers for authorization
	headers := map[string][]string{
		"Authorization": {"u1"},
	}

	// Connect to WebSocket server using Gorilla WebSocket
	conn, _, err := websocket.DefaultDialer.Dial(u.String(), headers)
	if err != nil {
		log.Fatal("connection fail", err)
	}
	defer conn.Close() // Close connection when function ends

	slog.Info("connected to gorilla websocket server")

	// send random message to server every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			<-ticker.C
			message := fmt.Sprintf("hello gorilla server %d", rand.IntN(10))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(message)); err != nil {
				log.Fatal(err)
			}
		}
	}()

	// receive messages from server
	for {
		_, message, err := conn.ReadMessage()
		if err != nil {
			slog.Warn("fail to receive", "err", err)
			break // quit loop
		}

		slog.Info("message received", "message", string(message))

		if string(message) == "ping" {
			pongMessage := fmt.Sprintf("pong %d", rand.IntN(10))
			if err := conn.WriteMessage(websocket.TextMessage, []byte(pongMessage)); err != nil {
				log.Fatal(err)
			}
		}
	}

	fmt.Println("gorilla client disconnected")
}