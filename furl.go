package furl

import (
	"math/rand"
	"net/http"
	"path"
	"sync"
	"time"
)

const (
	defaultKeyLength = 6
	defaultRetries   = 100
	maxURLLength     = 2048
)

func allValid(_ string) bool {
	return true
}

type loader interface {
	Load() (map[string]string, error)
	Save(key, url string) error
}

func save(_, _ string) error {
	return nil
}

type Furl struct {
	urlValidator, keyValidator func(string) bool
	keyLength, retries         uint
	save                       func(string, string) error
	rand                       *rand.Rand

	mu   sync.RWMutex
	urls map[string]string
}

func New(opts ...Option) *Furl {
	f := &Furl{
		urlValidator: allValid,
		keyValidator: allValid,
		keyLength:    defaultKeyLength,
		retries:      defaultRetries,
		save:         save,
	}
	for _, o := range opts {
		o(f)
	}
	if f.urls == nil {
		f.urls = make(map[string]string)
	}
	if f.rand == nil {
		f.rand = rand.New(rand.NewSource(time.Now().UnixMicro()))
	}
	return f
}

func (f *Furl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		f.get(w, r)
	case http.MethodPost:
		f.post(w, r)
	case http.MethodOptions:
		f.options(w, r)
	}
}

func (f *Furl) get(w http.ResponseWriter, r *http.Request) {
	key := path.Base(r.URL.Path)
	f.mu.RLock()
	url, ok := f.urls[key]
	f.mu.RUnlock()
	if ok {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	} else {
		http.NotFound(w, r)
	}
}

func (f *Furl) post(w http.ResponseWriter, r *http.Request) {
}

func (f *Furl) options(w http.ResponseWriter, r *http.Request) {
	key := path.Base(r.URL.Path)
	if key == "" {
		w.Header().Add("Allow", "OPTIONS, POST")
		return
	}
	f.mu.RLock()
	_, ok := f.urls[key]
	f.mu.RUnlock()
	if ok {
		w.Header().Add("Allow", "OPTIONS, GET, HEAD")
	} else {
		w.Header().Add("Allow", "OPTIONS, POST")
	}
}
