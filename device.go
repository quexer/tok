package tok

type Device interface {
	Uid() interface{}
	DvId() interface{}
	GetMeta(string) string
	PutMeta(string, string)
}

type device struct {
	uid  interface{}
	dvId string
	meta syncMap
}

func (p *device) Uid() interface{} {
	return p.uid
}
func (p *device) DvId() interface{} {
	return p.dvId
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
