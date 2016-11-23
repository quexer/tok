package tok

import (
	"expvar"
	"log"
)

var (
	expOnline = expvar.NewInt("tokOnline")
	expUp     = expvar.NewInt("tokUp")
	expDown   = expvar.NewInt("tokDown")
	expEnq    = expvar.NewInt("tokEnq")
	expDeq    = expvar.NewInt("tokDeq")
)

type checkFrame struct {
	uid    interface{}
	chBool chan bool //channel to return online status
}

type downFrame struct {
	uid   interface{}
	ttl   uint32
	data  []byte
	chErr chan error //channel to read send result from
}

type upFrame struct {
	dv   *Device
	data []byte
}

//HubConfig config struct for creating new Hub
type HubConfig struct {
	Actor Actor //Actor implement dispatch logic
	Q     Queue //Message Queue, if nil, message to offline user will not be cached
	Sso   bool  //If it's true, new connection  with same uid will kick off old ones
}

//Hub core of tok, dispatch message between connections
type Hub struct {
	sso           bool
	actor         Actor
	q             Queue
	cons          map[interface{}][]*connection //connection list
	chUp          chan *upFrame
	chDown        chan *downFrame
	chConState    chan *conState
	chReadSignal  chan interface{}
	chKick        chan interface{}
	chQueryOnline chan chan []interface{}
	chCheck       chan *checkFrame
}

func createHub(actor Actor, q Queue, sso bool) *Hub {
	if ReadTimeout > 0 {
		log.Println("[tok] read timeout is enabled, make sure it's greater than your client ping interval. otherwise you'll get read timeout err")
	} else {
		if actor.Ping() == nil {
			log.Fatalln("[tok] both read timeout and server ping have been disabled, server socket resource leak might happen")
		}
	}

	hub := &Hub{
		sso:           sso,
		actor:         actor,
		q:             q,
		cons:          make(map[interface{}][]*connection),
		chUp:          make(chan *upFrame),
		chDown:        make(chan *downFrame),
		chConState:    make(chan *conState),
		chReadSignal:  make(chan interface{}),
		chKick:        make(chan interface{}),
		chQueryOnline: make(chan chan []interface{}),
		chCheck:       make(chan *checkFrame),
	}
	go hub.run()
	return hub
}

func (p *Hub) run() {
	for {

		select {
		case state := <-p.chConState:
			//			log.Printf("connection state change: %v, %v \n", state.online, &state.con)

			if state.online {
				p.goOnline(state.con)
			} else {
				p.goOffline(state.con)
			}
			count := int64(len(p.cons))
			expOnline.Set(count)
		case f := <-p.chUp:
			//			log.Println("up data")
			expUp.Add(1)
			go func() {
				b, err := p.actor.BeforeReceive(f.dv, f.data)
				if err != nil {
					return
				}
				if b == nil {
					b = f.data
				}
				p.actor.OnReceive(f.dv, b)
			}()
		case ff := <-p.chDown:
			if l := p.cons[ff.uid]; len(l) > 0 {
				//online
				go p.down(ff, l)
			} else {
				//offline
				if ff.ttl == 0 {
					ff.chErr <- ErrOffline
					close(ff.chErr)
				} else {
					go p.cache(ff)
				}
			}
		case cf := <-p.chCheck:
			_, ok := p.cons[cf.uid]
			cf.chBool <- ok
			close(cf.chBool)
		case uid := <-p.chReadSignal:
			//only pop msg for online user
			if len(p.cons[uid]) > 0 {
				go p.popMsg(uid)
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

func (p *Hub) popMsg(uid interface{}) {
	if p.q == nil {
		return
	}

	for {
		b, err := p.q.Deq(uid)
		if err != nil {
			log.Println("deq error", err)
			return
		}
		if len(b) == 0 {
			//no more data in queue
			return
		}
		expDeq.Add(1)
		if err := p.Send(uid, b, 0); err != nil {
			if err := p.q.Enq(uid, b); err != nil {
				log.Println("re-cache err", err, uid)
			}
			return
		}
	}
}

//Send message to someone.
//ttl is expiry seconds. 0 means only send to online user
//If ttl = 0 and user is offline, ErrOffline will be returned.
//If ttl > 0 and user is offline or online but send fail, message will be cached for ttl seconds.
func (p *Hub) Send(to interface{}, b []byte, ttl uint32) error {

	ff := &downFrame{uid: to, data: b, ttl: ttl, chErr: make(chan error)}
	p.chDown <- ff
	err := <-ff.chErr
	if ttl > 0 && err != nil {
		//online send err
		ff.chErr = make(chan error) //create new channel
		go p.cache(ff)
		return <-ff.chErr
	}
	return err
}

//CheckOnline return whether user online or not
func (p *Hub) CheckOnline(uid interface{}) bool {
	cf := &checkFrame{uid: uid, chBool: make(chan bool)}
	p.chCheck <- cf
	return <-cf.chBool
}

//Online query online user list
func (p *Hub) Online() []interface{} {
	ch := make(chan []interface{})
	p.chQueryOnline <- ch
	return <-ch
}

func (p *Hub) cache(ff *downFrame) {
	defer close(ff.chErr)
	expEnq.Add(1)
	if p.q == nil {
		ff.chErr <- ErrQueueRequired
		return
	}

	if err := p.q.Enq(ff.uid, ff.data, ff.ttl); err != nil {
		ff.chErr <- err
	}
}

func (p *Hub) down(f *downFrame, conns []*connection) {
	expDown.Add(1)

	for _, con := range conns {
		b, err := p.actor.BeforeSend(con.dv, f.data)
		if err != nil {
			return
		}
		if b == nil {
			b = f.data
		}

		if con == nil {
			log.Println("hub.down, conn is nil")
			continue
		}
		if err := con.Write(b); err != nil {
			f.chErr <- err
			return
		}
		go p.actor.OnSent(con.dv, f.data)
	}

}

func (p *Hub) goOffline(conn *connection) {
	l := p.cons[conn.uid()]
	rest := connExclude(l, conn)

	//this connection has gotten offline, ignore
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

func (p *Hub) byeThenClose(conn *connection) {
	b := p.actor.Bye(conn.dv, "sso")
	if b != nil {
		data, err := p.actor.BeforeSend(conn.dv, b)
		if err == nil {
			if data != nil {
				b = data
			}
			conn.Write(b)
		}
	}
	p.close(conn)
}

func (p *Hub) close(conn *connection) {
	conn.close()
	p.actor.OnClose(conn.dv)
}

func (p *Hub) goOnline(conn *connection) {
	defer func() {
		go p.tryDeliver(conn.uid())
	}()

	l := p.cons[conn.uid()]
	if l == nil {
		p.cons[conn.uid()] = []*connection{conn}
		return
	}

	if p.sso {
		for _, c := range l {
			//notify before close connection
			go p.byeThenClose(c)
		}
		p.cons[conn.uid()] = []*connection{conn}
		return
	}

	//it's a new connection
	if len(connExclude(l, conn)) == len(l) {
		l = append(l, conn)
		p.cons[conn.uid()] = l
	}
}

//tryDeliver try to deliver all messages, if uid is online
func (p *Hub) tryDeliver(uid interface{}) {
	p.chReadSignal <- uid
}

//Kick kick all connections of uid
func (p *Hub) Kick(uid interface{}) {
	p.chKick <- uid
}

func (p *Hub) stateChange(conn *connection, online bool) {
	p.chConState <- &conState{conn, online}
}

//receive data from user
func (p *Hub) receive(dv *Device, b []byte) {
	p.chUp <- &upFrame{dv: dv, data: b}
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
