package tui

import "time"

type cacheEntry struct {
	items  []listItem
	expiry time.Time // zero means no expiry
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
	if !ok {
		return nil, false
	}
	if !e.expiry.IsZero() && time.Now().After(e.expiry) {
		return nil, false
	}
	return e.items, true
}

// Pin stores items under key with no expiry — they persist until explicitly invalidated.
func (c *listCache) Pin(key string, items []listItem) {
	c.entries[key] = cacheEntry{items: items}
}

// Invalidate removes the entry for key so the next Get is a miss.
func (c *listCache) Invalidate(key string) {
	delete(c.entries, key)
}
