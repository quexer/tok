/**
 * tcp connection adapter
 */

package tok

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"io"
	"log"
	"net"
	"net/http"
	"time"
)

const (
	tcp_header_len = 4
)

var (
	TCP_MAX_PACK_LEN uint32 = 4 * 1024 * 1024 //upper limit for single message
)

type tcpAdapter struct {
	conn net.Conn
}

func (p *tcpAdapter) Read() ([]byte, error) {
	//read header
	b := make([]byte, tcp_header_len)
	if _, err := io.ReadFull(p.conn, b); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	var n uint32
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	if n > TCP_MAX_PACK_LEN {
		return nil, fmt.Errorf("pack length %dM can't greater than %dM", n/1024/1024, TCP_MAX_PACK_LEN/1024/1024)
	}

	b = make([]byte, n)
	_, err := io.ReadFull(p.conn, b)
	return b, err

}

func (p *tcpAdapter) Write(b []byte) error {
	n := uint32(len(b))

	buf := new(bytes.Buffer)

	if err := binary.Write(buf, binary.BigEndian, &n); err != nil {
		return err
	}
	_, err := p.conn.Write(append(buf.Bytes(), b...))

	return err
}

func (p *tcpAdapter) Close() {
	p.conn.Close()
}

//Create hub with hubConfig, then create Tcp listener on addr
//return hub
func Listen(hubConfig *HubConfig, addr string) (*Hub, error) {
	hub := createHub(hubConfig.Actor, hubConfig.Q, hubConfig.Sso)
	return hub, ListenWithHub(hub, addr)
}

//Create Tcp listener with existing hub.
//return error if create listener failed
func ListenWithHub(hub *Hub, addr string) error {

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return err
	}

	initAuth := func(adapter ConAdapter) <-chan bool {
		chOk := make(chan bool, 1)
		go func() {
			b, err := adapter.Read()
			if err != nil {
				adapter.Close()
				chOk <- false
				return
			}
			r := &http.Request{Header: http.Header{"Cookie": {string(b)}}}
			uid, err := hub.actor.Auth(r)
			if err != nil {
				log.Println("401", err)
				adapter.Write(hub.actor.Bye("unauthorized"))
				adapter.Close()
				chOk <- false
				return
			}
			go initConnection(uid, adapter, hub)
			chOk <- true
		}()
		return chOk
	}

	initWithTimeout := func(conn net.Conn) {
		//		log.Println("raw tcp connection", conn.RemoteAddr())
		adapter := &tcpAdapter{conn: conn}
		select {
		case <-time.After(time.Second * 5):
			log.Println("init connection: timeout")
			adapter.Close()
		case <-initAuth(adapter):
		}
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting", err)
				continue
			}
			go initWithTimeout(conn)
		}
	}()

	return nil
}
