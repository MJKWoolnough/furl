package furl

import (
	"net/http"
	"sync"
)

type Furl struct {
	mu   sync.RWMutex
	urls map[string]string
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
