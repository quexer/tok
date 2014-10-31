/**
 *  queue reference implementation, using redis
 */
package ri

import (
	"fmt"
	"github.com/garyburd/redigo/redis"
	"github.com/quexer/tok"
	"log"
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
		TestOnBorrow: func(c redis.Conn, t time.Time) error {
			_, err := c.Do("PING")
			return err
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

	_, err := c.Do("RPUSH", name, data)
	if err != nil {
		return err
	}
	if len(ttl) > 0 && ttl[0] > 0 {
		_, err := c.Do("EXPIRE", name, ttl[0])
		if err != nil {
			log.Println("[warning] expire err", err)
		}
	}

	//	log.Println("enq", r)
	return nil
}

func (p *queue) Deq(uid interface{}) ([]byte, error) {
	c := p.pool.Get()
	defer c.Close()

	name := qname(uid)

	b, err := redis.Bytes(c.Do("LPOP", name))

	if err != nil && err != redis.ErrNil {
		return nil, err
	}

	return b, nil
}

func CreateRedisQ(server, auth string) tok.Queue {
	pool := createPool(server, auth)
	return &queue{pool: pool}
}
