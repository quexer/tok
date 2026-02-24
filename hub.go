package tok

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"time"

	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

type checkFrame struct {
	uid    any       // user id
	chBool chan bool // channel to return online status
}

type downFrame struct {
	uid   any        // user id
	ttl   uint32     // ttl in seconds
	data  []byte     // data to send
	chErr chan error // channel to read send result from
}

type upFrame struct {
	dv   *Device // user device
	data []byte  // data
}

// Hub core of tok, dispatch message between connections
type Hub struct {
	cons          map[any][]*connection // connection list
	chUp          chan *upFrame
	chDown        chan *downFrame
	chConState    chan *conState
	chReadSignal  chan any
	chKick        chan any
	chQueryOnline chan chan []any
	chCheck       chan *checkFrame
	config        *HubConfig         // config for hub
	ctx           context.Context    // hub lifecycle context
	cancel        context.CancelFunc // cancel function to trigger shutdown
	done          chan struct{}      // closed when run() exits
	inst          *instruments       // OTel instruments
}

func createHub(ctx context.Context, config *HubConfig) (*Hub, error) {
	if config.readTimeout > 0 {
		slog.Info("[tok] read timeout is enabled, make sure it's greater than your client ping interval. otherwise you'll get read timeout err")
	} else {
		// quit if both read timeout and ping are disabled
		if config.pingProducer == nil {
			return nil, errors.New("[tok] fatal: both read timeout and server ping have been disabled, server socket resource leak might happen")
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	hub := &Hub{
		cons:          make(map[any][]*connection),
		chUp:          make(chan *upFrame),
		chDown:        make(chan *downFrame),
		chConState:    make(chan *conState),
		chReadSignal:  make(chan any),
		chKick:        make(chan any),
		chQueryOnline: make(chan chan []any),
		chCheck:       make(chan *checkFrame),
		config:        config,
		ctx:           ctx,
		cancel:        cancel,
		done:          make(chan struct{}),
		inst:          newInstruments(config.meterProvider, config.tracerProvider),
	}
	go hub.run()
	return hub, nil
}

// Close gracefully shuts down the hub.
// It closes all managed connections, triggers close handlers, and stops the run loop.
// Close blocks until the run loop has fully exited.
// It is safe to call Close multiple times.
func (p *Hub) Close() {
	p.cancel()
	<-p.done

	// Close the queue if it implements io.Closer (e.g. MemoryQueue)
	if c, ok := p.config.q.(interface{ Close() }); ok {
		c.Close()
	}
}

func (p *Hub) run() {
	defer close(p.done)

	for {
		select {
		case <-p.ctx.Done():
			// graceful shutdown: close all connections
			for _, conns := range p.cons {
				for _, conn := range conns {
					p.close(conn)
				}
			}
			p.cons = make(map[any][]*connection)
			return
		case state := <-p.chConState:
			slog.Debug("connection state change", "online", state.online, "con", &state.con)

			prevCount := int64(len(p.cons))
			if state.online {
				p.goOnline(state.con)
			} else {
				p.goOffline(state.con)
			}
			delta := int64(len(p.cons)) - prevCount
			if delta != 0 {
				p.inst.connOnline.Add(p.ctx, delta)
			}
		case f := <-p.chUp:
			slog.Debug("up data")
			p.inst.messagesUp.Add(p.ctx, 1)
			go func() {
				_, span := p.inst.tracer.Start(context.Background(), "tok.Receive")
				defer span.End()

				data := f.data
				if hdl := p.config.hdlBeforeReceive; hdl != nil {
					if b, err := hdl.BeforeReceive(f.dv, f.data); err != nil {
						slog.Error("before receive failed", "err", err)
						span.SetStatus(codes.Error, err.Error())
						span.RecordError(err)
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
					go p.cache(p.ctx, ff)
				}
			}
		case cf := <-p.chCheck:
			_, ok := p.cons[cf.uid]
			cf.chBool <- ok
			close(cf.chBool)
		case uid := <-p.chReadSignal:
			// only pop msg for online user
			if len(p.cons[uid]) > 0 {
				go p.popMsg(p.ctx, uid)
			}
		case uid := <-p.chKick:
			p.innerKick(uid)
		case chOnline := <-p.chQueryOnline:
			result := make([]any, 0, len(p.cons))
			for uid := range p.cons {
				result = append(result, uid)
			}
			chOnline <- result
			close(chOnline)
		}
	}
}

func (p *Hub) popMsg(ctx context.Context, uid any) {
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
		p.inst.queueDeq.Add(ctx, 1)
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
// If ttl > 0 and user is offline, message will be cached for ttl seconds.
// If ttl > 0 and user is online, cache behavior on send failure depends on HubConfig PartialSendPolicy.
// Returns ErrHubClosed if the hub has been closed.
func (p *Hub) Send(ctx context.Context, to any, b []byte, ttl uint32) error {
	ctx, span := p.inst.tracer.Start(ctx, "tok.Send",
		trace.WithAttributes(attribute.Int("ttl", int(ttl))))
	defer span.End()
	start := time.Now()
	defer func() {
		p.inst.sendDuration.Record(ctx, time.Since(start).Seconds())
	}()

	// Use mctx (not shadowing ctx) because p.cache below needs the original caller ctx.
	mctx, cancel := p.withHubCtx(ctx)
	defer cancel()

	ff := &downFrame{uid: to, data: b, ttl: ttl, chErr: make(chan error, 1)}
	select {
	case p.chDown <- ff:
	case <-mctx.Done():
		err := context.Cause(mctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}

	var err error
	select {
	case err = <-ff.chErr:
	case <-mctx.Done():
		err = context.Cause(mctx)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}

	// if cache failed, return err directly
	if errors.Is(err, ErrCacheFailed) {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		return err
	}

	if ttl > 0 && err != nil && p.shouldCacheOnSendError(err) {
		// Create a new downFrame for caching to avoid channel reuse issues
		cacheFF := &downFrame{
			uid:   ff.uid,
			data:  ff.data,
			ttl:   ff.ttl,
			chErr: make(chan error, 1),
		}
		// Use the original caller ctx so cache can finish even if hub is closing
		go p.cache(ctx, cacheFF)

		select {
		case cacheErr := <-cacheFF.chErr:
			if cacheErr != nil {
				span.SetStatus(codes.Error, cacheErr.Error())
				span.RecordError(cacheErr)
			}
			return cacheErr
		case <-mctx.Done():
			err = context.Cause(mctx)
			span.SetStatus(codes.Error, err.Error())
			span.RecordError(err)
			return err
		}
	}
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
	}
	return err
}

func (p *Hub) shouldCacheOnSendError(err error) bool {
	switch p.config.partialSendPolicy {
	case PartialSendCacheOnAnyFailure:
		return true
	case PartialSendCacheWhenAllFailed:
		return !errors.Is(err, ErrPartialDelivered)
	case PartialSendNoCache:
		return false
	default:
		return true
	}
}

// CheckOnline return whether user online or not.
// Returns false if the hub has been closed.
func (p *Hub) CheckOnline(ctx context.Context, uid any) bool {
	ctx, cancel := p.withHubCtx(ctx)
	defer cancel()

	cf := &checkFrame{uid: uid, chBool: make(chan bool, 1)}
	select {
	case p.chCheck <- cf:
	case <-ctx.Done():
		return false
	}

	select {
	case online := <-cf.chBool:
		return online
	case <-ctx.Done():
		return false
	}
}

// Online query online user list.
// Returns nil if the hub has been closed.
func (p *Hub) Online(ctx context.Context) []any {
	ctx, cancel := p.withHubCtx(ctx)
	defer cancel()

	ch := make(chan []any, 1)
	select {
	case p.chQueryOnline <- ch:
	case <-ctx.Done():
		return nil
	}

	select {
	case result := <-ch:
		return result
	case <-ctx.Done():
		return nil
	}
}

func (p *Hub) cache(ctx context.Context, ff *downFrame) {
	ctx, span := p.inst.tracer.Start(ctx, "tok.Cache")
	defer span.End()
	defer close(ff.chErr)
	p.inst.queueEnq.Add(ctx, 1)
	if p.config.q == nil {
		err := fmt.Errorf("%w: %w", ErrCacheFailed, ErrQueueRequired)
		span.SetStatus(codes.Error, err.Error())
		span.RecordError(err)
		ff.chErr <- err
		return
	}

	if err := p.config.q.Enq(ctx, ff.uid, ff.data, ff.ttl); err != nil {
		wrapped := fmt.Errorf("%w: %w", ErrCacheFailed, err)
		span.SetStatus(codes.Error, wrapped.Error())
		span.RecordError(wrapped)
		ff.chErr <- wrapped
	}
}

func (p *Hub) down(f *downFrame, conns []*connection) {
	defer close(f.chErr)
	p.inst.messagesDown.Add(context.Background(), 1)

	var lastErr error
	var sent int
	var failed int
	for _, con := range conns {
		data, err := p.beforeSend(con.dv, f.data)
		if err != nil {
			lastErr = err
			failed++
			continue
		}
		if err := con.Write(data); err != nil {
			lastErr = err
			failed++
			continue
		}
		sent++

		if hdl := p.config.hdlAfterSend; hdl != nil {
			go hdl.AfterSend(con.dv, f.data)
		}
	}

	if failed == 0 {
		f.chErr <- nil
		return
	}
	if sent > 0 {
		f.chErr <- fmt.Errorf("%w: %w", ErrPartialDelivered, lastErr)
		return
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

func (p *Hub) innerKick(uid any) {
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
		return
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
		go p.tryDeliver(p.ctx, conn.uid())
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
func (p *Hub) tryDeliver(ctx context.Context, uid any) {
	select {
	case p.chReadSignal <- uid:
	case <-p.ctx.Done():
	}
}

// Kick all connections of uid.
// No-op if the hub has been closed.
func (p *Hub) Kick(ctx context.Context, uid any) {
	ctx, cancel := p.withHubCtx(ctx)
	defer cancel()

	select {
	case p.chKick <- uid:
	case <-ctx.Done():
	}
}

func (p *Hub) stateChange(conn *connection, online bool) {
	select {
	case p.chConState <- &conState{conn, online}:
	case <-p.ctx.Done():
	}
}

// receive data from user
func (p *Hub) receive(dv *Device, b []byte) {
	select {
	case p.chUp <- &upFrame{dv: dv, data: b}:
	case <-p.ctx.Done():
	}
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

// withHubCtx merges the hub's lifecycle context with a caller context.
// The returned context is cancelled when either context is done.
// Use context.Cause(ctx) to get the cancellation reason:
//   - ErrHubClosed if the hub was shut down
//   - the caller context's error otherwise
func (p *Hub) withHubCtx(ctx context.Context) (context.Context, context.CancelFunc) {
	ctx, cancel := context.WithCancelCause(ctx)
	stop := context.AfterFunc(p.ctx, func() {
		cancel(ErrHubClosed)
	})
	return ctx, func() { stop(); cancel(nil) }
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
