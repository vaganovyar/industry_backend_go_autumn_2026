package main

import (
	"container/list"
	"sync"
)

type LRU[K comparable, V any] interface {
	Get(key K) (value V, ok bool)
	Set(key K, value V)
}
type entry[K comparable, V any] struct {
	key   K
	value V
}
type LRUCache[K comparable, V any] struct {
	capacity int
	mu       sync.Mutex
	ll       list.List
	items    map[K]*list.Element
}

func NewLRUCache[K comparable, V any](capacity int) *LRUCache[K, V] {
	items := make(map[K]*list.Element, capacity)
	lruCache := &LRUCache[K, V]{
		capacity: capacity,
		ll:       list.List{},
		items:    items,
	}
	lruCache.ll.Init()

	return lruCache
}
func (c *LRUCache[K, V]) Get(key K) (value V, ok bool) {
	if c.capacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	item, ok := c.items[key]
	if ok {
		c.ll.MoveToFront(item)
		entryValue := item.Value.(entry[K, V])
		value = entryValue.value
	}

	return value, ok
}
func (c *LRUCache[K, V]) Set(key K, value V) {
	if c.capacity <= 0 {
		return
	}

	c.mu.Lock()
	defer c.mu.Unlock()

	_, isExists := c.items[key]
	if !isExists {
		element := c.ll.PushFront(entry[K, V]{key, value})
		c.items[key] = element
		currentCapacity := len(c.items)
		if currentCapacity > c.capacity {
			oldestElement := c.ll.Back()
			delete(c.items, oldestElement.Value.(entry[K, V]).key)
			c.ll.Remove(oldestElement)
		}
	} else {
		item := c.items[key]
		entryValue := item.Value.(entry[K, V])
		entryValue.value = value
		item.Value = entryValue
	}
}
