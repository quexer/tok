/**
 * hub, core of chat application
 */

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
	expErr    = expvar.NewInt("tokErr")
)

type fatFrame struct {
	frame *frame //frame to be sent
	ttl   uint32
	chErr chan error //channel to read send result from
}

type frame struct {
	uid  interface{}
	data []byte
}

//Config to create new Hub
type HubConfig struct {
	Actor Actor //Actor implement dispatch logic
	Q     Queue //Message Queue, if nil, message to offline user will not be cached
	Sso   bool  //If it's true, new connection  with same uid will kick off old ones
}

//Dispatch message between connections
type Hub struct {
	sso           bool
	actor         Actor
	q             Queue
	cons          map[interface{}][]*connection //connection list
	chUp          chan *frame
	chDown        chan *fatFrame //for online user
	chDown2       chan *fatFrame //for all user
	chConState    chan *conState
	chReadSignal  chan interface{}
	chQueryOnline chan chan []interface{}
}

func createHub(actor Actor, q Queue, sso bool) *Hub {
	if READ_TIMEOUT > 0 {
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
		chUp:          make(chan *frame),
		chDown:        make(chan *fatFrame),
		chDown2:       make(chan *fatFrame),
		chConState:    make(chan *conState),
		chReadSignal:  make(chan interface{}),
		chQueryOnline: make(chan chan []interface{}),
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
			go p.actor.OnReceive(f.uid, f.data)
		case ff := <-p.chDown:
			if len(p.cons[ff.frame.uid]) > 0 {
				go p.down(ff, p.cons[ff.frame.uid])
			} else {
				func() {
					defer func() {
						if err := recover(); err != nil {
							log.Println("bug:", err)
						}
					}()
					ff.chErr <- ErrOffline
					close(ff.chErr)
				}()
			}
		case ff := <-p.chDown2:
			if len(p.cons[ff.frame.uid]) > 0 {
				go p.down(ff, p.cons[ff.frame.uid])
			} else {
				go p.cache(ff)
			}

		case uid := <-p.chReadSignal:
			//only pop msg for online user
			if len(p.cons[uid]) > 0 {
				go p.popMsg(uid)
			}
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
		if err := p.Send(uid, b, -1); err != nil {
			log.Println("send err after deq")
			return
		}
	}
}

//Send message to someone,
//ttl is expiry seconds. 0 means forever
//if ttl >= 0 and user is offline, message will be cached for ttl seconds
//if ttl < 0 and user is offline, ErrOffline will be returned
//if ttl >=0 and user is online, but error occurred during send, message will be cached
//if ttl < 0 and user is online, but error occurred during send, the error will be returned
func (p *Hub) Send(to interface{}, b []byte, ttl ...int) error {
	t := -1
	if len(ttl) > 0 {
		t = ttl[0]
	}

	ff := &fatFrame{frame: &frame{uid: to, data: b}, ttl: uint32(t), chErr: make(chan error)}
	if t > 0 {
		p.chDown2 <- ff
	} else {
		p.chDown <- ff
	}
	err := <-ff.chErr
	if err == nil {
		return nil
	}
	expErr.Add(1)
	if t < 0 {
		return err
	}

	ff = &fatFrame{frame: &frame{uid: to, data: b}, ttl: uint32(t), chErr: make(chan error)}
	go p.cache(ff)
	return <-ff.chErr
}

//Query online user list
func (p *Hub) Online() []interface{} {
	ch := make(chan []interface{})
	p.chQueryOnline <- ch
	return <-ch
}

func (p *Hub) cache(ff *fatFrame) {
	defer close(ff.chErr)
	expEnq.Add(1)
	if p.q == nil {
		ff.chErr <- ErrQueueRequired
		return
	}

	f := ff.frame
	if err := p.q.Enq(f.uid, f.data, ff.ttl); err != nil {
		ff.chErr <- err
	}

	go p.actor.OnCache(f.uid)
}

func (p *Hub) down(ff *fatFrame, conns []*connection) {
	defer func(){
		if ff == nil{
			log.Println("defer hub.down, ff is nil")
			return
		}
		close(ff.chErr)
	}()
	expDown.Add(1)

	if ff == nil{
		log.Println("hub.down, ff is nil")
		return
	}
	if ff.frame == nil{
		log.Println("hub.down, ff.frame is nil")
		return
	}

	count := 0
	for _, con := range conns {
		if con == nil{
			log.Println("hub.down, conn is nil")
			continue
		}
		if err := con.Write(ff.frame.data); err != nil {
			ff.chErr <- err
			return
		}
		count += 1
	}

	go p.actor.OnSent(ff.frame.uid, ff.frame.data, count)
}

func (p *Hub) goOffline(conn *connection) {
	l := p.cons[conn.uid]
	rest := connExclude(l, conn)

	//this connection has gotten offline, ignore
	if len(l) == len(rest) {
		return
	}

	if len(rest) == 0 {
		delete(p.cons, conn.uid)
	} else {
		p.cons[conn.uid] = rest
	}

	go func(active int) {
		conn.close()
		p.actor.OnClose(conn.uid, active)
	}(len(rest))
}

func (p *Hub) goOnline(conn *connection) {
	l := p.cons[conn.uid]
	if l == nil {
		l = []*connection{conn}
	} else {
		if p.sso {
			for _, old := range l {
				//				log.Printf("kick %v\n", old)
				//notify before close connection
				go func() {
					old.Write(p.actor.Bye("sso"))
					old.close()
					//after sso kick, only one connection keep active
					p.actor.OnClose(conn.uid, 1)
				}()
			}
			l = []*connection{conn}
		} else {
			exists := false
			for _, c := range l {
				if c == conn {
					log.Println("warning, repeat online: ", c)
					exists = true
					break
				}
			}
			if !exists {
				l = append(l, conn)
			}
		}
	}
	p.cons[conn.uid] = l
	go p.TryDeliver(conn.uid)
}

//try to deliver all messages, if uid is online
func (p *Hub) TryDeliver(uid interface{}) {
	p.chReadSignal <- uid
}

func (p *Hub) stateChange(conn *connection, online bool) {
	p.chConState <- &conState{conn, online}
}

//receive data from user
func (p *Hub) receive(uid interface{}, b []byte) {
	p.chUp <- &frame{uid: uid, data: b}
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
