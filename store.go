package furl

import "sync"

// The Store interface allows for setting a custom storage solution to Furl,
// such as a database or keystore.
//
// The Get method should be used to retireve the URL associated with the passed
// key.
//
// The Tx method should start a thread safe writing context that will be used
// for creating new keys. See the Tx interface for more details.
type Store interface {
	Get(key string) (string, bool)
	Tx(func(tx Tx))
}

// The Tx interface represents a thread safe writing context for generating and
// storing keys and their corresponding URLs.
//
// The Has method may be called multiple times per Store.Tx call.
//
// The Set method will be called at most one time per Store.Tx call, and will
// be used to set the uniquely generated or passed key and its corresponding
// URL. The implementation of this method can be used to provide a more
// permanent storage for the key:url store.
type Tx interface {
	Has(key string) bool
	Set(key, url string)
}

// The StoreOption type is used to specify optional params to the NewStore
// function call.
type StoreOption func(*mapStore)

// The Data StoreOption is used to set the initial map of keys -> urls. The
// passed map should not be accessed by anything other than Furl until Furl is
// no longer is use.
//
// NB: Neither the keys or URLs are checked to be valid.
func Data(data map[string]string) StoreOption {
	return func(m *mapStore) {
		m.urls = data
	}
}

// The Save StoreOption is used to set a function that stores the keys and urls
// outside of Furl. For example, could be used to write to a file that be later
// loaded to provide the data for a future instance of Furl.
func Save(save func(key, url string)) StoreOption {
	return func(m *mapStore) {
		m.save = save
	}
}

func noSave(_, _ string) {}

// NewStore creates a map based implementation of the Store interface, with the
// following defaults that can be changed by adding StoreOption params:
//
// urls: By default, the Store is created with an empty map. This can be changed
// with the Data StoreOption.
//
// save: By default, there is no permanent storage of the key:url map. This can
// be changed by the Save StoreOption.
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
