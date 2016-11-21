package tok

type Device interface {
	Uid() interface{}
	Id() string
	GetMeta(string) string
	PutMeta(string, string)
}

//CreateDevice uid is user id, id is uuid of this device(could be empty)
func CreateDevice(uid interface{}, id string) Device {
	return &device{uid: uid, id: id}
}

type device struct {
	uid  interface{}
	id   string
	meta syncMap
}

func (p *device) Uid() interface{} {
	return p.uid
}
func (p *device) Id() string {
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
