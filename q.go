package tok

//Queue is FIFO queue interface, used by Hub
type Queue interface {
	Enq(uid interface{}, data []byte, ttl ...uint32) error
	Deq(uid interface{}) ([]byte, error)
	Len(uid interface{}) (int, error)
}
