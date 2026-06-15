package app

// In-process replacement for cache_put_data/cache_get_data (Load.php).
// PHP used memcached/APC across share-nothing workers; a mutex-guarded map
// with TTLs does the same job inside one Go process.

import (
	"sync"
	"time"
)

type cacheEntry struct {
	value   any
	expires time.Time
}

type memCache struct {
	mu sync.RWMutex
	m  map[string]cacheEntry
}

func newMemCache() *memCache {
	return &memCache{m: make(map[string]cacheEntry)}
}

func (c *memCache) Put(key string, value any, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if value == nil {
		delete(c.m, key)
		return
	}
	c.m[key] = cacheEntry{value, time.Now().Add(ttl)}
}

func (c *memCache) Get(key string) (any, bool) {
	c.mu.RLock()
	e, ok := c.m[key]
	c.mu.RUnlock()
	if !ok || time.Now().After(e.expires) {
		return nil, false
	}
	return e.value, true
}
