/**
 * hub, core of chat application
 */

package tok

import (
	"expvar"
	"fmt"
	"log"
	"strings"
)

var (
	expOnline = expvar.NewInt("tokOnline")
	expUp     = expvar.NewInt("tokUp")
	expDown   = expvar.NewInt("tokDown")
	expEnq    = expvar.NewInt("tokEnq")
	expDeq    = expvar.NewInt("tokDeq")
)

type frame struct {
	uid  interface{}
	data []byte
}

//Config to create new Hub
type HubConfig struct {
	Actor Actor //Actor implement dispatch logic
	Q     Queue //Message Q, if nil, a memory based queue will be used
	Sso   bool  //If it's true, new connection will kick off old ones with same uid
}

//Dispatch message between connections
type Hub struct {
	sso           bool
	actor         Actor
	q             Queue
	cons          map[interface{}][]*connection //connection list
	chUp          chan *frame
	chDown        chan *frame //for online user
	chDown2       chan *frame //for all user
	chConState    chan *conState
	chReadSignal  chan interface{}
	chQueryOnline chan chan []interface{}
}

func createHub(actor Actor, q Queue, sso bool) *Hub {
	if q == nil {
		q = CreateMemQ()
	}
	hub := &Hub{
		sso:          sso,
		actor:        actor,
		q:            q,
		cons:         make(map[interface{}][]*connection),
		chUp:         make(chan *frame),
		chDown:       make(chan *frame),
		chDown2:      make(chan *frame),
		chConState:   make(chan *conState),
		chReadSignal: make(chan interface{}, 100),
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
			fmt.Printf("%stok online:%8d", strings.Repeat("\b", 19), count)
		case f := <-p.chUp:
			//			log.Println("up data")
			expUp.Add(1)
			go p.actor.OnReceive(f.uid, f.data)
		case f := <-p.chDown:
			expDown.Add(1)
			p.down(f)
		case f := <-p.chDown2:
			if len(p.cons[f.uid]) > 0 {
				expDown.Add(1)
				p.down(f)
			} else {
				expEnq.Add(1)
				go p.cache(f)
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
	for {
		b, err := p.q.Deq(uid)
		if err != nil {
			log.Println("deq error", err)
			return
		}
		if b == nil {
			//no more data in queue
			return
		}
		expDeq.Add(1)
		p.chDown <- &frame{uid: uid, data: b}
	}
}

//Send message to someone
func (p *Hub) Send(to interface{}, b []byte, cacheIfOffline bool) {
	f := &frame{uid: to, data: b}
	if cacheIfOffline {
		p.chDown2 <- f
	} else {
		p.chDown <- f
	}
}

//Query online user list
func (p *Hub) Online() []interface{} {
	ch := make(chan []interface{})
	p.chQueryOnline <- ch
	return <-ch
}

func (p *Hub) cache(f *frame) {
	if err := p.q.Enq(f.uid, f.data); err != nil {
		log.Println("send binary to q err ", err)
	}
}

func (p *Hub) down(f *frame) {
	for _, con := range p.cons[f.uid] {
		con.ch <- f.data
	}
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

	conn.close()
}

func (p *Hub) goOnline(conn *connection) {
	l := p.cons[conn.uid]
	if l == nil {
		l = []*connection{conn}
	} else {
		if p.sso {
			for _, old := range l {
				log.Printf("kick %v\n", old)
				//notify before close connection
				old.ch <- p.actor.Bye("sso")
				old.close()
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
	p.startRead(conn.uid)
}

func (p *Hub) startRead(uid interface{}) {
	p.chReadSignal <- uid
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
