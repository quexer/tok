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
	// TCPMaxPackLen upper limit for single message
	TCPMaxPackLen uint32 = 4 * 1024 * 1024
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

	// read header
	b := make([]byte, tcpHeaderLen)
	if _, err := io.ReadFull(p.conn, b); err != nil {
		return nil, err
	}

	buf := bytes.NewBuffer(b)
	var n uint32
	if err := binary.Read(buf, binary.BigEndian, &n); err != nil {
		return nil, err
	}

	if n > TCPMaxPackLen {
		return nil, fmt.Errorf("pack length %dM can't greater than %dM", n/1024/1024, TCPMaxPackLen/1024/1024)
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
	// set write deadline
	if err := p.conn.SetWriteDeadline(time.Now().Add(WriteTimeout)); err != nil {
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

func (p *tcpAdapter) Close() error {
	return p.conn.Close()
}

func (p *tcpAdapter) ShareConn(adapter conAdapter) bool {
	tcpAdp, ok := adapter.(*tcpAdapter)
	if !ok {
		return false
	}
	return p.conn == tcpAdp.conn
}

// Listen create Tcp listener with hub.
// If config is not nil, a new hub will be created and replace the old one.
// addr is the tcp address to be listened on.
// auth function is used for user authorization
// return error if listen failed.
func Listen(hub *Hub, config *HubConfig, addr string, auth TCPAuthFunc) (*Hub, error) {
	if config != nil {
		hub = createHub(config)
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
		if err := conn.SetReadDeadline(time.Now().Add(config.authTimeout)); err != nil {
			log.Println("set auth deadline err: ", err)
			_ = conn.Close()
			return
		}

		adapter := &tcpAdapter{conn: conn, readTimeout: config.authTimeout}
		b, err := adapter.Read()
		if err != nil {
			//			log.Println("tcp auth err ", err)
			_ = adapter.Close()
			return
		}

		dv, err := auth(b)
		if err != nil {
			log.Printf("tcp auth err: %+v", err)
			_ = adapter.Close()
			return
		}

		if ReadTimeout > 0 {
			adapter.readTimeout = ReadTimeout
		} else {
			adapter.readTimeout = 0
		}

		hub.initConnection(dv, adapter)
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

// TCPAuthFunc tcp auth function
// parameter is the first package content of connection. return Device interface
type TCPAuthFunc func([]byte) (*Device, error)
