package auth

import (
	"sync"
	"time"

	"github.com/MG-RAST/AWE/lib/user"
)

type cache struct {
	sync.RWMutex
	m map[string]cacheValue
}

type cacheValue struct {
	expires time.Time
	user    *user.User
}

func (c *cache) lookup(header string) *user.User {
	v, ok := c.get(header)
	if ok {
		if time.Now().Before(v.expires) {
			return v.user
		} else {
			c.delete(header)
		}
	}
	return nil
}

func (c *cache) add(header string, u *user.User) {
	c.Lock()
	defer c.Unlock()
	c.m[header] = cacheValue{
		expires: time.Now().Add(1 * time.Hour),
		// expires: time.Now().Add(time.Duration(conf.AUTH_CACHE_TIMEOUT) * time.Minute),
		user: u,
	}
	return
}

func (c *cache) get(header string) (v cacheValue, ok bool) {
	c.RLock()
	defer c.RUnlock()
	v, ok = c.m[header]
	return
}
func (c *cache) delete(header string) {
	c.Lock()
	defer c.Unlock()
	delete(c.m, header)
	return
}
