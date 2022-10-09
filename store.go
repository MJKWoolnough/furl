package furl

import "sync"

type Store interface {
	Get(key string) (string, bool)
	Tx(func(tx Tx))
}

type Tx interface {
	Has(key string) bool
	Set(key, url string)
}

type StoreOption func(*mapStore)

func Data(data map[string]string) StoreOption {
	return func(m *mapStore) {
		m.urls = data
	}
}

func Save(save func(key, url string)) StoreOption {
	return func(m *mapStore) {
		m.save = save
	}
}

func noSave(_, _ string) {}

func NewStore(opts ...StoreOption) Store {
	m := &mapStore{
		save: noSave,
	}
	for _, o := range opts {
		o(m)
	}
	if m.urls == nil {
		m.urls = make(map[string]string)
	}
	return m
}

type mapStore struct {
	mu   sync.RWMutex
	urls map[string]string
	save func(string, string)
}

func (m *mapStore) Get(key string) (string, bool) {
	m.mu.RLock()
	url, ok := m.urls[key]
	m.mu.RUnlock()
	return url, ok
}

func (m *mapStore) Tx(fn func(tx Tx)) {
	m.mu.Lock()
	fn(m)
	m.mu.Unlock()
}

func (m *mapStore) Has(key string) bool {
	_, ok := m.urls[key]
	return ok
}

func (m *mapStore) Set(key, url string) {
	m.urls[key] = url
	m.save(key, url)
}
