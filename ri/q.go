/**
 *  queue reference implementation, using redis
 */
package ri

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/quexer/tok"
	"time"
)

func createPool(server, auth string) *redis.Pool {
	return &redis.Pool{
		MaxIdle:     3,
		IdleTimeout: 240 * time.Second,
		Dial: func() (redis.Conn, error) {
			c, err := redis.Dial("tcp", server)
			if err != nil {
				return nil, err
			}
			if auth != "" {
				if _, err := c.Do("AUTH", auth); err != nil {
					c.Close()
					return nil, err
				}
			}
			return c, err
		},
	}
}

func qname(uid interface{}) string {
	return fmt.Sprintf("q%v", uid)
}

type queue struct {
	pool *redis.Pool
}

func (p *queue) Len(uid interface{}) (int, error) {
	c := p.pool.Get()
	defer c.Close()

	name := qname(uid)

	i, err := redis.Int(c.Do("LLEN", name))

	if err != nil && err.Error() == "redigo: nil returned" {
		//expire
		return 0, nil
	}

	return i, err
}

func (p *queue) Enq(uid interface{}, data []byte, ttl ...uint32) error {
	c := p.pool.Get()
	defer c.Close()

	name := qname(uid)
	c.Send("MULTI")
	c.Send("RPUSH", name, data)
	if len(ttl) > 0 && ttl[0] > 0 {
		c.Send("EXPIRE", name, ttl[0])
	}
	_, err := c.Do("EXEC")

	//	log.Println("enq", r)
	return err
}

func (p *queue) Deq(uid interface{}) ([]byte, error) {
	c := p.pool.Get()
	defer c.Close()

	name := qname(uid)
	c.Send("MULTI")
	c.Send("LPOP", name)
	r, err := redis.Values(c.Do("EXEC"))

	if err != nil && err != redis.ErrNil {
		return nil, err
	}
	b, err := redis.Bytes(r[0], err)
	if err != nil && err != redis.ErrNil {
		return nil, err
	}

	return b, nil
}

func CreateRedisQ(server, auth string) tok.Queue {
	pool := createPool(server, auth)
	return &queue{pool: pool}
}
