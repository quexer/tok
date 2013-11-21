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
	TCP_HEADER_LEN   = 4
	TCP_MAX_PACK_LEN = 4 * 1024 * 1024
)

type tcpAdapter struct {
	conn net.Conn
}

func (p *tcpAdapter) Read() ([]byte, error) {
	//read header
	b := make([]byte, TCP_HEADER_LEN)
	if _, err := io.ReadFull(p.conn, b); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	var n uint32
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	if n > TCP_MAX_PACK_LEN {
		return nil, fmt.Errorf("may be error pack length: %dk", n/1024/1024)
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

	if _, err := p.conn.Write(buf.Bytes()); err != nil {
		return err
	}

	_, err := p.conn.Write(b)
	return err
}

func (p *tcpAdapter) Close() {
	p.conn.Close()
}

func CreateTcpListener(auth Auth, hub *Hub, addr string) error {
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
			uid, err := auth(r)
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
