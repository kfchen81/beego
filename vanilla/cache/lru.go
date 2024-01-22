package cache

import (
	"container/list"
	"github.com/kfchen81/beego/metrics"
	"sync"
	"time"
)

type entry struct {
	key     interface{}
	value   interface{}
	element *list.Element
	expires time.Time
	timer   *time.Timer
}

type Option func(*cache)

type EvictCallback func(key interface{}, value interface{})

func WithTTL(val time.Duration) Option {
	return func(c *cache) {
		c.ttl = val
	}
}

func WithEvictCallBack(callback EvictCallback) Option {
	return func(c *cache) {
		c.onEvict = callback
	}
}

func WithReset() Option {
	return func(c *cache) {
		c.NoReset = true
	}
}

func WithValue2Key() Option {
	return func(c *cache) {
		c.enableValue2Key = true
	}
}

// cache is the type that implements the ttlru
type cache struct {
	name            string
	cap             int
	ttl             time.Duration
	items           map[interface{}]*entry
	evictList       *list.List
	lock            sync.RWMutex
	NoReset         bool
	onEvict         EvictCallback
	enableValue2Key bool
	value2Key       map[interface{}]interface{}
}

// New creates a new Cache with cap entries that expire after ttl has
// elapsed since the item was added, modified or accessed.
func NewLRUCache(name string, cap int, opts ...Option) Cache {
	c := cache{cap: cap, name: name}

	for _, opt := range opts {
		opt(&c)
	}

	if c.cap <= 0 || c.ttl < 0 {
		return nil
	}

	metrics.GetLRUCacheCounter().WithLabelValues(c.name, "evict").Inc()
	c.items = make(map[interface{}]*entry, cap)
	if c.enableValue2Key {
		c.value2Key = make(map[interface{}]interface{}, cap)
	}
	c.evictList = list.New()
	metrics.GetLRUCacheCounter().DeleteLabelValues(c.name)
	return &c
}

func (c *cache) Set(key, value interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	// Check for existing item
	if ent, ok := c.items[key]; ok {
		c.updateEntry(ent, value)
		return false
	}

	// Evict oldest if next entry would exceed capacity
	evict := c.evictList.Len() == c.cap
	if evict {
		if ele := c.evictList.Back(); ele != nil {
			ent := ele.Value.(*entry)
			metrics.GetLRUCacheCounter().WithLabelValues(c.name, "evict").Inc()
			c.removeEntry(ent)
		}
	}

	c.insertEntry(key, value)

	metrics.GetLRUCacheCounter().WithLabelValues(c.name, "set").Inc()
	return evict
}

func (c *cache) insertEntry(key, value interface{}) *entry {
	// must already have a write lock

	ent := &entry{
		key:     key,
		value:   value,
		expires: time.Now().Add(c.ttl),
	}
	// push *entry to element and get *element
	ele := c.evictList.PushFront(ent)
	// set *element to *entry.element, so can delete element
	// from list when entry time expired
	ent.element = ele

	if c.ttl > 0 {
		ent.timer = time.AfterFunc(c.ttl, func() {
			c.lock.Lock()
			defer c.lock.Unlock()
			c.removeEntry(ent)
		})
	}

	c.items[key] = ent
	if c.enableValue2Key {
		c.value2Key[value] = key
	}
	return ent
}

func (c *cache) updateEntry(e *entry, value interface{}) {
	// must already have a write lock

	if c.enableValue2Key {
		delete(c.value2Key, e.value)
		c.value2Key[value] = e.key
	}
	// update with the new value
	e.value = value

	// reset the ttl
	c.renewEntry(e, true)
}

func (c *cache) resetEntryTTL(e *entry) {
	// must already have a write lock

	// reset the expiration timer
	if c.ttl > 0 {
		e.timer.Reset(c.ttl)
	}

	// set the new expiration time
	e.expires = time.Now().Add(c.ttl)
}

func (c *cache) renewEntry(e *entry, reset bool) {
	if reset {
		c.resetEntryTTL(e)
	}
	c.evictList.MoveToFront(e.element)
}

func (c *cache) removeEntry(e *entry) {
	// must already have a write lock
	// delete the item from the map
	delete(c.items, e.key)
	if c.enableValue2Key {
		delete(c.value2Key, e.value)
	}
	if e.element != nil {
		c.evictList.Remove(e.element)
		e.element = nil // avoid memory leaks
	}
	if c.onEvict != nil {
		c.onEvict(e.key, e.value)
	}
}

func (c *cache) Get(key interface{}) (interface{}, bool) {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if c.ttl == 0 || time.Now().Before(ent.expires) {
			c.renewEntry(ent, !c.NoReset)
			metrics.GetLRUCacheCounter().WithLabelValues(c.name, "get-hit").Inc()
			return ent.value, true
		}
	}
	metrics.GetLRUCacheCounter().WithLabelValues(c.name, "get-miss").Inc()
	return nil, false
}

func (c *cache) Keys() []interface{} {
	c.lock.RLock()
	defer c.lock.RUnlock()

	keys := make([]interface{}, 0, len(c.items))
	for k, v := range c.items {
		// the item should be automatically removed when it expires, but we
		// check just to be safe
		if c.ttl == 0 || time.Now().Before(v.expires) {
			keys = append(keys, k)
		}
	}

	return keys
}

func (c *cache) Len() int {
	c.lock.RLock()
	defer c.lock.RUnlock()
	return c.evictList.Len()
}

func (c *cache) Cap() int {
	return c.cap
}

func (c *cache) Purge() {
	c.lock.Lock()
	defer c.lock.Unlock()

	for _, ent := range c.items {
		if c.onEvict != nil {
			c.onEvict(ent.key, ent.value)
		}
	}
	c.evictList.Init()
	c.items = make(map[interface{}]*entry, c.cap)
	if c.enableValue2Key {
		c.value2Key = make(map[interface{}]interface{}, c.cap)
	}
}

func (c *cache) Del(key interface{}) bool {
	c.lock.Lock()
	defer c.lock.Unlock()

	if ent, ok := c.items[key]; ok {
		c.removeEntry(ent)
		return true
	}

	return false
}

func (c *cache) getKeyByValue(value interface{}) interface{} {
	if !c.enableValue2Key {
		return nil
	}
	c.lock.RUnlock()
	defer c.lock.RUnlock()
	return c.value2Key[value]
}

func (c *cache) DelByValue(value interface{}) bool {
	key := c.getKeyByValue(value)
	if key == nil {
		return false
	}
	return c.Del(key)
}
