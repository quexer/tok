package tok

import "sync"

type syncMap struct {
	sync.RWMutex
	m map[interface{}]interface{}
}

func (p *syncMap) Put(key, val interface{}) {
	p.Lock()
	defer p.Unlock()

	if p.m == nil {
		p.m = map[interface{}]interface{}{}
	}
	p.m[key] = val
}

func (p *syncMap) Remove(key interface{}) {
	p.Lock()
	defer p.Unlock()

	if p.m == nil {
		return
	}
	delete(p.m, key)
}

func (p *syncMap) Clear() {
	p.Lock()
	defer p.Unlock()

	p.m = nil
}

func (p *syncMap) Len() int {
	p.RLock()
	defer p.RUnlock()

	return len(p.m)
}

func (p *syncMap) Get(key interface{}) (interface{}, bool) {
	p.RLock()
	defer p.RUnlock()

	if p.m == nil {
		return nil, false
	}
	val, ok := p.m[key]
	return val, ok
}
