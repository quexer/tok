package tok

import (
	"context"
	"errors"
	"expvar"
	"fmt"
	"log"
	"log/slog"
	"time"
)

var (
	expOnline = expvar.NewInt("tokOnline")
	expUp     = expvar.NewInt("tokUp")
	expDown   = expvar.NewInt("tokDown")
	expEnq    = expvar.NewInt("tokEnq")
	expDeq    = expvar.NewInt("tokDeq")
)

type checkFrame struct {
	uid    interface{} // user id
	chBool chan bool   // channel to return online status
}

type downFrame struct {
	uid   interface{} // user id
	ttl   uint32      // ttl in seconds
	data  []byte      // data to send
	chErr chan error  // channel to read send result from
}

type upFrame struct {
	dv   *Device // user device
	data []byte  // data
}

// Hub core of tok, dispatch message between connections
type Hub struct {
	cons          map[interface{}][]*connection // connection list
	chUp          chan *upFrame
	chDown        chan *downFrame
	chConState    chan *conState
	chReadSignal  chan interface{}
	chKick        chan interface{}
	chQueryOnline chan chan []interface{}
	chCheck       chan *checkFrame
	config        *HubConfig // config for hub
}

func createHub(config *HubConfig) *Hub {
	if config.readTimeout > 0 {
		slog.Info("[tok] read timeout is enabled, make sure it's greater than your client ping interval. otherwise you'll get read timeout err")
	} else {
		// quit if both read timeout and ping are disabled
		if config.pingProducer == nil {
			log.Fatalln("[tok] fatal: both read timeout and server ping have been disabled, server socket resource leak might happen")
		}
	}

	hub := &Hub{
		cons:          make(map[interface{}][]*connection),
		chUp:          make(chan *upFrame),
		chDown:        make(chan *downFrame),
		chConState:    make(chan *conState),
		chReadSignal:  make(chan interface{}),
		chKick:        make(chan interface{}),
		chQueryOnline: make(chan chan []interface{}),
		chCheck:       make(chan *checkFrame),
		config:        config,
	}
	go hub.run()
	return hub
}

func (p *Hub) run() {
	for {

		select {
		case state := <-p.chConState:
			slog.Debug("connection state change", "online", state.online, "con", &state.con)

			if state.online {
				p.goOnline(state.con)
			} else {
				p.goOffline(state.con)
			}
			count := int64(len(p.cons))
			expOnline.Set(count)
		case f := <-p.chUp:
			slog.Debug("up data")
			expUp.Add(1)
			go func() {
				// default is f.data
				data := f.data
				// Use the optional BeforeReceive handler if provided
				if hdl := p.config.hdlBeforeReceive; hdl != nil {
					if b, err := hdl.BeforeReceive(f.dv, f.data); err != nil {
						slog.Error("before receive failed", "err", err)
						return
					} else {
						data = b
					}
				}
				p.config.actor.OnReceive(f.dv, data)
			}()
		case ff := <-p.chDown:
			if l := p.cons[ff.uid]; len(l) > 0 {
				// online
				go p.down(ff, l)
			} else {
				// offline
				if ff.ttl == 0 {
					ff.chErr <- ErrOffline
					close(ff.chErr)
				} else {
					go p.cache(context.Background(), ff)
				}
			}
		case cf := <-p.chCheck:
			_, ok := p.cons[cf.uid]
			cf.chBool <- ok
			close(cf.chBool)
		case uid := <-p.chReadSignal:
			// only pop msg for online user
			if len(p.cons[uid]) > 0 {
				go p.popMsg(context.Background(), uid)
			}
		case uid := <-p.chKick:
			p.innerKick(uid)
		case chOnline := <-p.chQueryOnline:
			result := make([]interface{}, 0, len(p.cons))
			for uid := range p.cons {
				result = append(result, uid)
			}
			chOnline <- result
			close(chOnline)
		}
	}
}

func (p *Hub) popMsg(ctx context.Context, uid interface{}) {
	if p.config.q == nil {
		return
	}
	for {
		b, err := p.config.q.Deq(ctx, uid)
		if err != nil {
			slog.Warn("deq failed", "err", err)
			return
		}
		if len(b) == 0 {
			// no more data in queue
			return
		}
		expDeq.Add(1)
		if err := p.Send(ctx, uid, b, 0); err != nil {
			if err := p.config.q.Enq(ctx, uid, b); err != nil {
				slog.Warn("re-cache failed", "err", err, "uid", uid)
			}
			return
		}
	}
}

// Send message to someone.
// ttl is expiry seconds. 0 means only send to online user
// If ttl = 0 and user is offline, ErrOffline will be returned.
// If ttl > 0 and user is offline or online but send fail, message will be cached for ttl seconds.
func (p *Hub) Send(ctx context.Context, to interface{}, b []byte, ttl uint32) error {

	ff := &downFrame{uid: to, data: b, ttl: ttl, chErr: make(chan error)}
	p.chDown <- ff
	err := <-ff.chErr

	// if cache failed, return err directly
	if errors.Is(err, ErrCacheFailed) {
		return err
	}

	if ttl > 0 && err != nil {
		// Create a new downFrame for caching to avoid channel reuse issues
		cacheFF := &downFrame{
			uid:   ff.uid,
			data:  ff.data,
			ttl:   ff.ttl,
			chErr: make(chan error),
		}
		// Use the passed context instead of Background()
		go p.cache(ctx, cacheFF)
		return <-cacheFF.chErr
	}
	return err
}

// CheckOnline return whether user online or not
func (p *Hub) CheckOnline(ctx context.Context, uid interface{}) bool {
	cf := &checkFrame{uid: uid, chBool: make(chan bool)}
	p.chCheck <- cf
	return <-cf.chBool
}

// Online query online user list
func (p *Hub) Online(ctx context.Context) []interface{} {
	ch := make(chan []interface{})
	p.chQueryOnline <- ch
	return <-ch
}

func (p *Hub) cache(ctx context.Context, ff *downFrame) {
	defer close(ff.chErr)
	expEnq.Add(1)
	if p.config.q == nil {
		ff.chErr <- fmt.Errorf("%w: %w", ErrCacheFailed, ErrQueueRequired)
		return
	}

	if err := p.config.q.Enq(ctx, ff.uid, ff.data, ff.ttl); err != nil {
		ff.chErr <- fmt.Errorf("%w: %w", ErrCacheFailed, err)
	}
}

func (p *Hub) down(f *downFrame, conns []*connection) {
	defer close(f.chErr)
	expDown.Add(1)

	var lastErr error
	for _, con := range conns {
		data, err := p.beforeSend(con.dv, f.data)
		if err != nil {
			lastErr = err
			continue
		}
		if err := con.Write(data); err != nil {
			lastErr = err
			continue
		}

		if hdl := p.config.hdlAfterSend; hdl != nil {
			go hdl.AfterSend(con.dv, f.data)
		}
	}
	f.chErr <- lastErr
}

func (p *Hub) goOffline(conn *connection) {
	l := p.cons[conn.uid()]
	rest := connExclude(l, conn)

	// this connection has gotten offline, ignore
	if len(l) == len(rest) {
		return
	}

	if len(rest) == 0 {
		delete(p.cons, conn.uid())
	} else {
		p.cons[conn.uid()] = rest
	}

	go p.close(conn)
}

func (p *Hub) innerKick(uid interface{}) {
	for _, conn := range p.cons[uid] {
		go p.close(conn)
	}
	delete(p.cons, uid)
}

func (p *Hub) byeThenClose(kicker *Device, conn *connection) {
	defer p.close(conn)

	// Only generate bye message if ByeGenerator is configured
	if p.config.byeGenerator == nil {
		return
	}

	byeData := p.config.byeGenerator.Bye(kicker, "sso", conn.dv)
	if byeData == nil {
		return
	}

	data, err := p.beforeSend(conn.dv, byeData)
	if err != nil {
		slog.Warn("[tok] before send bye failed", "err", err)
	}
	if err := conn.Write(data); err != nil {
		slog.Warn("[tok] write bye failed", "err", err)
	}
}

func (p *Hub) close(conn *connection) {
	conn.close()

	// Call the optional close handler if configured
	if hdl := p.config.closeHandler; hdl != nil {
		hdl.OnClose(conn.dv)
	}
}

func (p *Hub) goOnline(conn *connection) {
	defer func() {
		go p.tryDeliver(context.Background(), conn.uid())
	}()

	l := p.cons[conn.uid()]
	if l == nil {
		p.cons[conn.uid()] = []*connection{conn}
		return
	}

	if p.config.sso {
		for _, c := range l {
			if conn.ShareConn(c) {
				continue // never close share connection
			}
			// notify before close connection
			go p.byeThenClose(conn.dv, c)
		}
		p.cons[conn.uid()] = []*connection{conn}
		return
	}

	// it's a new connection
	if len(connExclude(l, conn)) == len(l) {
		l = append(l, conn)
		p.cons[conn.uid()] = l
	}
}

// tryDeliver try to deliver all messages, if uid is online
func (p *Hub) tryDeliver(ctx context.Context, uid interface{}) {
	p.chReadSignal <- uid
}

// Kick all connections of uid
func (p *Hub) Kick(ctx context.Context, uid interface{}) {
	p.chKick <- uid
}

func (p *Hub) stateChange(conn *connection, online bool) {
	p.chConState <- &conState{conn, online}
}

// receive data from user
func (p *Hub) receive(dv *Device, b []byte) {
	p.chUp <- &upFrame{dv: dv, data: b}
}

// RegisterConnection registers a custom connection with the hub.
// This method allows users to integrate their own connection types (e.g., QUIC, Unix sockets)
// by implementing the ConAdapter interface.
//
// Parameters:
//   - dv: The authenticated device information
//   - adapter: The connection adapter implementing the ConAdapter interface
//
// The connection will be managed by the hub and will receive messages sent to the device.
// The hub will handle connection lifecycle, including ping/pong if configured.
//
// Example:
//
//	adapter := &MyCustomAdapter{conn: customConn}
//	device := tok.CreateDevice("user123", "session456")
//	hub.RegisterConnection(device, adapter)
func (p *Hub) RegisterConnection(ctx context.Context, dv *Device, adapter ConAdapter) {
	// create context for this connection
	connCtx, cancel := context.WithCancel(ctx)

	conn := &connection{
		dv:         dv,
		adapter:    adapter,
		hub:        p,
		cancelFunc: cancel,
	}

	// change conn state to online
	p.stateChange(conn, true)

	// start server ping loop if necessary
	if p.config.pingProducer != nil {
		ticker := time.NewTicker(p.config.serverPingInterval)
		go func() {
			defer ticker.Stop()
			for {
				select {
				case <-connCtx.Done():
					return
				case <-ticker.C:
					// Use the optional BeforeSend function if provided
					// Get fresh ping data for each iteration to ensure the current state of the connection
					pingData := p.config.pingProducer.Ping()
					data, err := p.beforeSend(dv, pingData)
					if err != nil {
						slog.Warn("[tok] before send ping failed", "err", err)
						continue
					}
					if err := conn.Write(data); err != nil {
						slog.Warn("[tok] write ping failed", "err", err)
						// write failed, connection might be closed, exit ping loop
						return
					}
				}
			}
		}()
	}

	// block on read
	conn.readLoop()
}

// beforeSend preprocess outgoing data before sending it.
func (p *Hub) beforeSend(dv *Device, data []byte) ([]byte, error) {
	hdl := p.config.hdlBeforeSend
	if hdl == nil {
		return data, nil
	}
	return hdl.BeforeSend(dv, data)
}

func connExclude(l []*connection, ex *connection) []*connection {
	rest := make([]*connection, 0, len(l))
	for _, c := range l {
		if c != ex {
			rest = append(rest, c)
		}
	}
	return rest
}
