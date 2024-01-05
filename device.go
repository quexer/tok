package tok

import (
	"sync"
)

// CreateDevice uid is user id, id is uuid of this device(could be empty)
func CreateDevice(uid interface{}, id string) *Device {
	return &Device{uid: uid, id: id}
}

// Device device struct
type Device struct {
	uid  interface{}
	id   string
	meta sync.Map
}

// UID return user id
func (p *Device) UID() interface{} {
	return p.uid
}

// ID return device uuid(could be empty)
func (p *Device) ID() string {
	return p.id
}

// GetMeta return device meta
func (p *Device) GetMeta(key string) string {
	if v, ok := p.meta.Load(key); ok {
		return v.(string)
	}

	return ""
}

// PutMeta set device meta
func (p *Device) PutMeta(key string, val string) {
	p.meta.Store(key, val)
}
