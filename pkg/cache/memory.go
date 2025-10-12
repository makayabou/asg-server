package cache

import (
	"context"
	"sync"
	"time"
)

type memoryCache struct {
	items map[string]*memoryItem
	ttl   time.Duration

	mux sync.RWMutex
}

func NewMemory(ttl time.Duration) Cache {
	return &memoryCache{
		items: make(map[string]*memoryItem),
		ttl:   ttl,

		mux: sync.RWMutex{},
	}
}

type memoryItem struct {
	value      string
	validUntil time.Time
}

func newItem(value string, opts options) *memoryItem {
	item := &memoryItem{
		value:      value,
		validUntil: opts.validUntil,
	}

	return item
}

func (i *memoryItem) isExpired(now time.Time) bool {
	return !i.validUntil.IsZero() && now.After(i.validUntil)
}

// Cleanup implements Cache.
func (m *memoryCache) Cleanup(_ context.Context) error {
	m.cleanup(func() {})

	return nil
}

// Delete implements Cache.
func (m *memoryCache) Delete(_ context.Context, key string) error {
	m.mux.Lock()
	delete(m.items, key)
	m.mux.Unlock()

	return nil
}

// Drain implements Cache.
func (m *memoryCache) Drain(_ context.Context) (map[string]string, error) {
	var cpy map[string]*memoryItem

	m.cleanup(func() {
		cpy = m.items
		m.items = make(map[string]*memoryItem)
	})

	items := make(map[string]string, len(cpy))
	for key, item := range cpy {
		items[key] = item.value
	}

	return items, nil
}

// Get implements Cache.
func (m *memoryCache) Get(_ context.Context, key string) (string, error) {
	return m.getValue(func() (*memoryItem, bool) {
		m.mux.RLock()
		item, ok := m.items[key]
		m.mux.RUnlock()

		return item, ok
	})
}

// GetAndDelete implements Cache.
func (m *memoryCache) GetAndDelete(_ context.Context, key string) (string, error) {
	return m.getValue(func() (*memoryItem, bool) {
		m.mux.Lock()
		item, ok := m.items[key]
		delete(m.items, key)
		m.mux.Unlock()

		return item, ok
	})
}

// Set implements Cache.
func (m *memoryCache) Set(_ context.Context, key string, value string, opts ...Option) error {
	m.mux.Lock()
	m.items[key] = m.newItem(value, opts...)
	m.mux.Unlock()

	return nil
}

// SetOrFail implements Cache.
func (m *memoryCache) SetOrFail(_ context.Context, key string, value string, opts ...Option) error {
	m.mux.Lock()
	defer m.mux.Unlock()

	if item, ok := m.items[key]; ok {
		if !item.isExpired(time.Now()) {
			return ErrKeyExists
		}
	}

	m.items[key] = m.newItem(value, opts...)
	return nil
}

func (m *memoryCache) newItem(value string, opts ...Option) *memoryItem {
	o := options{
		validUntil: time.Time{},
	}
	if m.ttl > 0 {
		o.validUntil = time.Now().Add(m.ttl)
	}
	o.apply(opts...)

	return newItem(value, o)
}

func (m *memoryCache) getItem(getter func() (*memoryItem, bool)) (*memoryItem, error) {
	item, ok := getter()

	if !ok {
		return nil, ErrKeyNotFound
	}

	if item.isExpired(time.Now()) {
		return nil, ErrKeyExpired
	}

	return item, nil
}

func (m *memoryCache) getValue(getter func() (*memoryItem, bool)) (string, error) {
	item, err := m.getItem(getter)
	if err != nil {
		return "", err
	}

	return item.value, nil
}

func (m *memoryCache) cleanup(cb func()) {
	t := time.Now()

	m.mux.Lock()
	for key, item := range m.items {
		if item.isExpired(t) {
			delete(m.items, key)
		}
	}

	cb()
	m.mux.Unlock()
}
