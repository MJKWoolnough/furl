// Package Furl provides a drop-in http.Handler that provides short url
// redirects for longer URLs.
package furl

import (
	"encoding/base64"
	"encoding/json"
	"encoding/xml"
	"fmt"
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

var xmlStart = xml.StartElement{
	Name: xml.Name{
		Local: "furl",
	},
}

func allValid(_ string) bool {
	return true
}

func save(_, _ string) error {
	return nil
}

type httpError struct {
	Code  int
	Error string
}

var (
	failedKeyGenerationError = &httpError{
		Code:  http.StatusInternalServerError,
		Error: failedKeyGeneration,
	}
	invalidKeyError = &httpError{
		Code:  http.StatusUnprocessableEntity,
		Error: invalidKey,
	}
	keyExistsError = &httpError{
		Code:  http.StatusMethodNotAllowed,
		Error: keyExists,
	}
)

// The Furl type represents a keystore of URLs to either generated or supplied
// keys.
type Furl struct {
	urlValidator, keyValidator func(string) bool
	keyLength, retries         uint
	save                       func(string, string) error
	rand                       *rand.Rand

	mu   sync.RWMutex
	urls map[string]string
}

// The New function creates a new instance of Furl, with the following defaults
// that can be changed by adding Option params.
//
// urlValidator: By default all strings are treated as valid URLs, this can be
// changed by using the URLValidator Option.
//
// keyValidator: By default all strings are treated as valid Keys, this can be
// changed by using the KeyValidator Option.
//
// keyLength: The default length of generated keys (before base64 encoding) is
// 6 and can be changed by using the KeyLength Option.
//
// retries: The default number of retries the key generator will before
// increasing the key length is 100 and can be changed by using the
// CollisionRetries Option.
//
// save: By default no data is permanently stored and this can be changed by
// using the IOStore Option.
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

// The ServeHTTP method satifies the http.Handler interface and provides the
// following endpoints:
// GET /[key] -  Will redirect the call to the associated URL if it exists, or
//               will return 404 Not Found if it doesn't exists and 422
//               Unprocessable Entity if the key is invalid.
// POST / -      The root can be used to add urls to the store with a generated
//               key. The URL must be specified in the POST body as per the
//               specification below.
// POST /[key] - Will attempt to create the specified path with the URL
//               provided as below. If the key is invalid, will respond with
//               422 Unprocessable Entity. This method cannot be used on
//               existing keys.
//
// The URL for the POST methods can be provided in a few content types:
// application/json:                  {"key": "KEY HERE", "url": "URL HERE"}
// text/xml:                          <furl><key>KEY HERE</key><url>URL HERE</url></furl>
// application/x-www-form-urlencoded: key=KEY+HERE&url=URL+HERE
// text/plain:                        URL HERE
//
// For the json, xml, and form content types, the key can be ommitted if it has
// been supplied in the path or if the key is to be generated.
//
// The response type will be determined by the POST content type:
// application/json: {"key": "KEY HERE", "url": "URL HERE"}
// text/xml:         <furl><key>KEY HERE</key><url>URL HERE</url></furl>
// text/plain:       KEY HERE
//
// For application/x-www-form-urlencoded, the content type of the return will
// be text/html and the response will match that of text/plain.
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
	if !f.keyValidator(key) {
		http.Error(w, invalidKey, http.StatusUnprocessableEntity)
		return
	}
	f.mu.RLock()
	url, ok := f.urls[key]
	f.mu.RUnlock()
	if ok {
		http.Redirect(w, r, url, http.StatusMovedPermanently)
	} else {
		http.NotFound(w, r)
	}
}

func writeError(w http.ResponseWriter, status int, contentType, err string) {
	var format string
	switch contentType {
	case "text/json", "application/json":
		format = "{\"error\":%q}"
	case "text/xml", "application/xml":
		format = "<furl><error>%s</error></furl>"
	default:
		format = "%s"
	}
	w.WriteHeader(status)
	fmt.Fprintf(w, format, err)
}

type keyURL struct {
	Key string `json:"key" xml:"key"`
	URL string `json:"url" xml:"url"`
}

func (f *Furl) post(w http.ResponseWriter, r *http.Request) {
	var (
		data keyURL
		err  error
	)
	contentType := r.Header.Get("Content-Type")
	switch contentType {
	case "text/json", "application/json":
		json.NewDecoder(r.Body).Decode(&data)
	case "text/xml", "application/xml":
		err = xml.NewDecoder(r.Body).Decode(&data)
	case "application/x-www-form-urlencoded":
		err = r.ParseForm()
		data.Key = r.PostForm.Get("key")
		data.URL = r.PostForm.Get("url")
		contentType = "text/html"
	case "text/plain":
		var sb strings.Builder
		_, err = io.Copy(&sb, r.Body)
		data.URL = sb.String()
	default:
		http.Error(w, unrecognisedContentType, http.StatusUnsupportedMediaType)
		return
	}
	w.Header().Set("Content-Type", contentType)
	if err != nil {
		writeError(w, http.StatusBadRequest, contentType, failedReadRequest)
		return
	}
	if len(data.URL) > maxURLLength || data.URL == "" || !f.urlValidator(data.URL) {
		writeError(w, http.StatusBadRequest, contentType, invalidURL)
		return
	}
	if data.Key == "" {
		data.Key = path.Base("/" + r.URL.Path)
	}
	var herr *httpError
	f.mu.Lock()
	if data.Key == "" || data.Key == "/" || data.Key == "." || data.Key == ".." {
		herr = f.genKey(&data)
	} else {
		herr = f.setKey(&data)
	}
	if herr == nil {
		f.urls[data.Key] = data.URL
		f.save(data.Key, data.URL)
	}
	f.mu.Unlock()
	if herr != nil {
		writeError(w, herr.Code, contentType, herr.Error)
		return
	}
	switch contentType {
	case "text/json", "application/json":
		json.NewEncoder(w).Encode(data)
	case "text/xml", "application/xml":
		xml.NewEncoder(w).EncodeElement(data, xmlStart)
	case "text/html", "text/plain":
		io.WriteString(w, data.Key)
	}
}

func (f *Furl) genKey(data *keyURL) *httpError {
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
			return failedKeyGenerationError
		}
	}
	return nil
}

func (f *Furl) setKey(data *keyURL) *httpError {
	if len(data.Key) > maxKeyLength || !f.keyValidator(data.Key) {
		return invalidKeyError
	}
	if _, ok := f.urls[data.Key]; ok {
		return keyExistsError
	}
	return nil
}

func (f *Furl) options(w http.ResponseWriter, r *http.Request) {
	key := path.Base(r.URL.Path)
	if key == "" || key == "/" {
		w.Header().Add("Allow", optionsPost)
	} else if !f.keyValidator(key) {
		http.Error(w, invalidKey, http.StatusUnprocessableEntity)
		return
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
