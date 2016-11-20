package tok

type Device interface {
	Uid() interface{}
	Id() interface{}
	GetMeta(string) string
	PutMeta(string, string)
}

type device struct {
	uid  interface{}
	id   string
	meta syncMap
}

func (p *device) Uid() interface{} {
	return p.uid
}
func (p *device) Id() interface{} {
	return p.id
}

func (p *device) GetMeta(key string) string {
	if v, ok := p.meta.Get(key); ok {
		return v.(string)
	} else {
		return ""
	}

}
func (p *device) PutMeta(key string, val string) {
	p.meta.Put(key, val)
}
