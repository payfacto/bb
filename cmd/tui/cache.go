package tui

import "time"

const defaultCacheTTL = 5 * time.Minute

type cacheEntry struct {
	items  []listItem
	expiry time.Time
}

type listCache struct {
	entries map[string]cacheEntry
}

func newListCache() *listCache {
	return &listCache{entries: make(map[string]cacheEntry)}
}

// Get returns cached items for key. Returns false if the entry is missing or expired.
func (c *listCache) Get(key string) ([]listItem, bool) {
	e, ok := c.entries[key]
	if !ok || time.Now().After(e.expiry) {
		return nil, false
	}
	return e.items, true
}

func (c *listCache) Set(key string, items []listItem, ttl time.Duration) {
	c.entries[key] = cacheEntry{items: items, expiry: time.Now().Add(ttl)}
}

// Invalidate removes the entry for key so the next Get is a miss.
func (c *listCache) Invalidate(key string) {
	delete(c.entries, key)
}
