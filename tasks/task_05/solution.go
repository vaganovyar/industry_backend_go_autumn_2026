package main

type Cache[K comparable, V any] struct {
	capacity int
	items    map[K]V
}

func NewCache[K comparable, V any](capacity int) *Cache[K, V] {
	items := make(map[K]V, capacity)

	return &Cache[K, V]{capacity, items}
}
func (c *Cache[K, V]) Get(k K) (v V, ok bool) {
	item, ok := c.items[k]
	return item, ok
}
func (c *Cache[K, V]) Set(k K, v V) bool {
	_, isExists := c.items[k]
	if len(c.items) >= c.capacity && !isExists {
		return false
	}

	c.items[k] = v

	return true
}
