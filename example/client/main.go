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
	url    = "ws://localhost:8090/ws" // WebSocket 服务器地址
)

func main() {
	// 创建 WebSocket 配置
	config, err := websocket.NewConfig(url, origin)
	if err != nil {
		log.Fatal("配置创建失败:", err)
	}

	config.Header.Set("Authorization", "u1")

	// 连接到 WebSocket 服务器
	conn, err := websocket.DialConfig(config)
	if err != nil {
		log.Fatal("连接失败:", err)
	}
	defer conn.Close() // 确保最后关闭连接

	fmt.Println("已连接到 WebSocket 服务器")

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
			log.Println("接收消息失败:", err)
			break // 退出循环
		}

		fmt.Printf("收到消息: %s\n", message)

		if message == "ping" {
			if err := websocket.Message.Send(conn, fmt.Sprintf("pong %d\n", rand.IntN(10))); err != nil {
				log.Fatal(err)
			}
		}

	}

	fmt.Println("客户端已断开连接")
}
