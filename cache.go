package main

import (
	"sync"
	"time"
)

// cacheEntry menyimpan data beserta waktu kedaluwarsa.
type cacheEntry struct {
	value     any
	expiresAt time.Time
}

// Cache adalah struktur in-memory cache berbasis map yang aman dari race condition.
type Cache struct {
	mu         sync.RWMutex
	items      map[string]cacheEntry
	defaultTTL time.Duration
}

// NewCache membuat instans Cache baru dengan default TTL 30 detik bila ttl <= 0.
func NewCache(ttl time.Duration) *Cache {
	if ttl <= 0 {
		ttl = 30 * time.Second
	}
	return &Cache{
		items:      make(map[string]cacheEntry),
		defaultTTL: ttl,
	}
}

// Get mengambil nilai dari cache; kedaluwarsa dicek saat akses untuk menghindari overhead goroutine janitor latar belakang yang rumit.
func (c *Cache) Get(key string) (any, bool) {
	c.mu.RLock()
	item, ok := c.items[key]
	c.mu.RUnlock()

	if !ok {
		return nil, false
	}

	if time.Now().After(item.expiresAt) {
		c.mu.Lock()
		delete(c.items, key)
		c.mu.Unlock()
		return nil, false
	}

	return item.value, true
}

// Set menyimpan nilai ke dalam cache menggunakan durasi default TTL.
func (c *Cache) Set(key string, val any) {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.items[key] = cacheEntry{
		value:     val,
		expiresAt: time.Now().Add(c.defaultTTL),
	}
}
