package furl

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"io"
	"math/rand"
	"net/http"
	"path"
	"strings"
	"sync"
	"time"
)

const (
	defaultKeyLength = 6
	defaultRetries   = 100
	maxURLLength     = 2048
	maxKeyLength     = 2048

	unrecognisedContentType = "unrecognised content-type"
	failedReadRequest       = "failed to read request"
	invalidURL              = "invalid url"
	failedKeyGeneration     = "failed to generate key"
	invalidKey              = "invalid key"
	keyExists               = "key exists"

	optionsPost    = "OPTIONS, POST"
	optionsGetHead = "OPTIONS, GET, HEAD"
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
	var (
		data struct {
			Key string `json:"key" xml:"key"`
			URL string `json:"url" xml:"url"`
		}
		err error
	)
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "text/json", "application/json":
		json.NewDecoder(r.Body).Decode(&data)
	case "text/xml", "application/xml":
		err = xml.NewDecoder(r.Body).Decode(&data)
	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
		data.URL = r.PostForm.Get("url")
		contentType = "text/html"
	case "text/plain":
		var sb strings.Builder
		_, err = io.Copy(&sb, r.Body)
		data.URL = sb.String()
	default:
		http.Error(w, unrecognisedContentType, http.StatusBadRequest)
		return
	}
	if err != nil {
		http.Error(w, failedReadRequest, http.StatusBadRequest)
		return
	}
	if len(data.URL) > maxURLLength || data.URL == "" || !f.urlValidator(data.URL) {
		http.Error(w, invalidURL, http.StatusBadRequest)
		return
	}
	data.Key = path.Base(r.URL.Path)
	f.mu.Lock()
	defer f.mu.Unlock()
	if data.Key == "" {
	Loop:
		for idLength := f.keyLength; ; idLength++ {
			keyBytes := make([]byte, idLength)
			for i := uint(0); i < f.retries; i++ {
				f.rand.Read(keyBytes) // NB: will never error
				data.Key = base64.RawURLEncoding.EncodeToString(keyBytes)
				if _, ok := f.urls[data.Key]; !ok && f.keyValidator(data.Key) {
					break Loop
				}
			}
			if idLength == maxKeyLength {
				http.Error(w, failedKeyGeneration, http.StatusInternalServerError)
				return
			}
		}
	} else {
		if len(data.Key) > maxKeyLength || !f.keyValidator(data.Key) {
			http.Error(w, invalidKey, http.StatusBadRequest)
			return
		}
		if _, ok := f.urls[data.Key]; ok {
			http.Error(w, keyExists, http.StatusMethodNotAllowed)
			return
		}
	}
	f.urls[data.Key] = data.URL
	f.save(data.Key, data.URL)
	w.Header().Set("Content-Type", contentType)
	switch contentType {
	case "text/json", "application/json":
		json.NewEncoder(w).Encode(data)
	case "text/xml", "application/xml":
		json.NewEncoder(w).Encode(data)
	case "text/html", "text/plain":
		io.WriteString(w, data.Key)
	}
}

func (f *Furl) options(w http.ResponseWriter, r *http.Request) {
	key := path.Base(r.URL.Path)
	if key == "" {
		w.Header().Add("Allow", optionsPost)
	} else {
		f.mu.RLock()
		_, ok := f.urls[key]
		f.mu.RUnlock()
		if ok {
			w.Header().Add("Allow", optionsGetHead)
		} else {
			w.Header().Add("Allow", optionsPost)
		}
	}
	w.WriteHeader(http.StatusNoContent)
}
