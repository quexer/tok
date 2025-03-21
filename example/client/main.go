package main

import (
	"fmt"
	"log"
	"math/rand/v2"
	"time"

	"golang.org/x/net/websocket"
)

const (
	origin = "http://localhost/"
	url    = "ws://localhost:8090/ws" // WebSocket server address
)

func main() {
	// 创建 WebSocket 配置
	config, err := websocket.NewConfig(url, origin)
	if err != nil {
		log.Fatal("fail to create websocket config", err)
	}

	config.Header.Set("Authorization", "u1")

	// 连接到 WebSocket 服务器
	conn, err := websocket.DialConfig(config)
	if err != nil {
		log.Fatal("connection fail", err)
	}
	defer conn.Close() // close connection when function ends

	fmt.Println("connected to websocket server")

	// send random message to server every 5 seconds
	ticker := time.NewTicker(5 * time.Second)
	go func() {
		for {
			<-ticker.C
			if err := websocket.Message.Send(conn, fmt.Sprintf("Hello Server %d\n", rand.IntN(10))); err != nil {
				log.Fatal(err)
			}
		}

	}()

	// 持续接收消息
	for {
		var message string
		err := websocket.Message.Receive(conn, &message)
		if err != nil {
			log.Println("fail to receive", err)
			break // quit loop
		}

		fmt.Printf("message received %s\n", message)

		if message == "ping" {
			if err := websocket.Message.Send(conn, fmt.Sprintf("pong %d\n", rand.IntN(10))); err != nil {
				log.Fatal(err)
			}
		}

	}

	fmt.Println("client disconnected")
}
