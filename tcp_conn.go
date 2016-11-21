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
	"time"
)

const (
	tcpHeaderLen = 4
)

var (
	//TCP_MAX_PACK_LEN upper limit for single message
	TCP_MAX_PACK_LEN uint32 = 4 * 1024 * 1024
)

type tcpAdapter struct {
	conn        net.Conn
	readTimeout time.Duration
}

func (p *tcpAdapter) Read() ([]byte, error) {
	var deadline time.Time
	if p.readTimeout > 0 {
		deadline = time.Now().Add(p.readTimeout)
	} else {
		deadline = time.Time{}
	}
	if err := p.conn.SetReadDeadline(deadline); err != nil {
		log.Println("[warning] setting read deadline: ", err)
		return nil, err
	}

	//read header
	b := make([]byte, tcpHeaderLen)
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

	if p.readTimeout > 0 {
		deadline = time.Now().Add(p.readTimeout)
	} else {
		deadline = time.Time{}
	}
	if err := p.conn.SetReadDeadline(deadline); err != nil {
		log.Println("[warning] setting read deadline: ", err)
		return nil, err
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

//Listen create Tcp listener with hub.
//If config is not nil, a new hub will be created and replace the old one.
//addr is the tcp address to be listened on.
//return error if listen failed.
func Listen(hub *Hub, config *HubConfig, addr string, auth TcpAuthFunc) (*Hub, error) {
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

		adapter := &tcpAdapter{conn: conn, readTimeout: AUTH_TIMEOUT}
		b, err := adapter.Read()
		if err != nil {
			//			log.Println("tcp auth err ", err)
			adapter.Close()
			return
		}

		dv, err := auth(b)
		if err != nil {
			adapter.Close()
			return
		}

		if READ_TIMEOUT > 0 {
			adapter.readTimeout = READ_TIMEOUT
		} else {
			adapter.readTimeout = 0
		}

		initConnection(dv, adapter, hub)
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

//TcpAuthFunc tcp auth function
//parameter is the first package content of connection. return Device interface
type TcpAuthFunc func([]byte) (Device, error)
