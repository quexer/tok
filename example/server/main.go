package main

import (
	"fmt"
	"log"
	"net/http"
	"time"

	"github.com/quexer/tok"
)

var (
	hub *tok.Hub
)

func main() {
	var hdl http.Handler
	tok.ServerPingInterval = 2 * time.Second
	actor := &simpleActor{}
	hc := tok.NewHubConfig(actor)

	authFunc := func(r *http.Request) (*tok.Device, error) {
		return tok.CreateDevice(fmt.Sprintf("%p", r), ""), nil
	}

	hub, hdl = tok.CreateWsHandler(authFunc, tok.WithWsHandlerHubConfig(hc))

	http.Handle("/ws", hdl)

	err := http.ListenAndServe(":8090", nil)
	if err != nil {
		log.Fatalf("Error starting HTTP server: %v", err)
	}

}

type simpleActor struct {
}

func (p *simpleActor) BeforeReceive(dv *tok.Device, data []byte) ([]byte, error) {
	log.Printf("BeforeReceive %+v, %s", dv, data)
	return data, nil
}

func (p *simpleActor) OnReceive(dv *tok.Device, data []byte) {
	log.Printf("OnReceive %+v, %s", dv, data)
	return
}

func (p *simpleActor) BeforeSend(dv *tok.Device, data []byte) ([]byte, error) {
	log.Printf("BeforeSend %+v, %s", dv, data)
	return data, nil
}

func (p *simpleActor) OnSent(dv *tok.Device, data []byte) {
	log.Printf("OnSent %+v, %s", dv, data)
	return
}

func (p *simpleActor) OnClose(dv *tok.Device) {
	log.Printf("OnClose %+v", dv)
	return
}

func (p *simpleActor) Ping() []byte {
	log.Println("Ping")
	return []byte("ping")
}

func (p *simpleActor) Bye(kicker *tok.Device, reason string, dv *tok.Device) []byte {
	return nil
}
