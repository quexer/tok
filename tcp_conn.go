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
	TCP_MAX_PACK_LEN uint32 = 4 * 1024 * 1024 //Upper limit for single message
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
	//set write deadline
	if err := p.conn.SetWriteDeadline(time.Now().Add(WRITE_TIMEOUT)); err != nil {
		log.Println("[warning] setting write deadline fail: ", err)
		return err
	}

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

//Create Tcp listener with hub.
//If config is not nil, a new hub will be created and replace the old one.
//addr is the tcp address to be listened on.
//return error if listen failed.
func Listen(hub *Hub, config *HubConfig, addr string) (*Hub, error) {
	if config != nil {
		hub = createHub(config.Actor, config.Q, config.Sso)
	}

	if hub == nil {
		log.Fatal("hub is needed")
	}

	listener, err := net.Listen("tcp", addr)
	if err != nil {
		return nil, err
	}

	initAuth := func(conn net.Conn) {
		//		log.Println("raw tcp connection", conn.RemoteAddr())
		if err := conn.SetReadDeadline(time.Now().Add(AUTH_TIMEOUT)); err != nil {
			log.Println("set auth deadline err: ", err)
			conn.Close()
			return
		}

		adapter := &tcpAdapter{conn: conn}
		b, err := adapter.Read()
		if err != nil {
			log.Println("tcp auth err ", err)
			adapter.Close()
			return
		}
		r := &http.Request{Header: http.Header{"Cookie": {string(b)}}}
		uid, err := hub.actor.Auth(r)
		if err != nil {
			log.Println("401", err)
			adapter.Write(hub.actor.Bye("unauthorized"))
			adapter.Close()
			return
		}

		if err := conn.SetReadDeadline(time.Time{}); err != nil {
			log.Println("clear auth deadline err: ", err)
			adapter.Close()
			return
		}

		initConnection(uid, adapter, hub)
	}

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				log.Println("Error accepting", err)
				continue
			}

			go initAuth(conn)
		}
	}()

	return hub, nil
}
