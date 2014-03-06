package tok

import (
	"fmt"
	"sync"
)

func (p *memQueue) qname(uid interface{}) string {
	return fmt.Sprintf("q%v", uid)
}

type memQueue struct {
	sync.RWMutex
	m map[string][][]byte
}

func (p *memQueue) Len(uid interface{}) (int, error) {
	p.RLock()
	defer p.RUnlock()

	name := p.qname(uid)
	l := p.m[name]
	if l == nil {
		return 0, nil
	} else {
		return len(l), nil
	}
}

func (p *memQueue) Enq(uid interface{}, data []byte) error {
	p.Lock()
	defer p.Unlock()

	name := p.qname(uid)
	l := p.m[name]
	l = append(l, data)
	p.m[name] = l
	return nil
}

func (p *memQueue) Deq(uid interface{}) ([]byte, error) {
	p.Lock()
	defer p.Unlock()

	name := p.qname(uid)
	l := p.m[name]

	if len(l) == 0 {
		return nil, nil
	}

	d := l[0]
	l = l[1:]
	p.m[name] = l

	return d, nil
}

func CreateMemQ() Queue {
	return &memQueue{m: make(map[string][][]byte)}
}
