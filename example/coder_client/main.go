package main

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/coder/websocket"
)

func main() {
	ctx := context.Background()

	// Connect to the WebSocket server
	conn, _, err := websocket.Dial(ctx, "ws://localhost:8091/ws", nil)
	if err != nil {
		log.Fatalf("Failed to connect: %v", err)
	}
	defer conn.Close(websocket.StatusNormalClosure, "")

	fmt.Println("Connected to Coder WebSocket server")

	// Send messages in a goroutine
	go func() {
		ticker := time.NewTicker(3 * time.Second)
		defer ticker.Stop()

		for i := 0; ; i++ {
			select {
			case <-ticker.C:
				msg := fmt.Sprintf("Hello from Coder client %d", i)
				err := conn.Write(ctx, websocket.MessageText, []byte(msg))
				if err != nil {
					log.Printf("Failed to send message: %v", err)
					return
				}
				fmt.Printf("Sent: %s\n", msg)
			case <-ctx.Done():
				return
			}
		}
	}()

	// Read messages
	for {
		msgType, data, err := conn.Read(ctx)
		if err != nil {
			log.Printf("Failed to read message: %v", err)
			break
		}

		if msgType == websocket.MessageText {
			fmt.Printf("Received text message: %s\n", string(data))
		} else if msgType == websocket.MessageBinary {
			fmt.Printf("Received binary message: %v\n", data)
		}
	}
}
