package furl

import (
	"net/http"
	"sync"
)

const (
	defaultKeyLength = 6
	defaultRetries   = 100
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

	mu   sync.RWMutex
	urls map[string]string
}

func New() *Furl {
	return &Furl{
		urlValidator: allValid,
		keyValidator: allValid,
		keyLength:    defaultKeyLength,
		retries:      defaultRetries,
		save:         save,
		urls:         make(map[string]string),
	}
}

func (f *Furl) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.Method {
	case http.MethodGet, http.MethodHead:
		f.get(w, r)
	case http.MethodPost:
		f.post(w, r)
	case http.MethodOptions:
		f.options(w)
	}
}

func (f *Furl) get(w http.ResponseWriter, r *http.Request) {
}

func (f *Furl) post(w http.ResponseWriter, r *http.Request) {
}

func (f *Furl) options(w http.ResponseWriter) {
	w.Header().Add("Allow", "OPTIONS, GET, HEAD, POST")
}
