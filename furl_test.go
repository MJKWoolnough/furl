package furl

import (
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	f := New(MemStore(map[string]string{
		"AAA": "http://www.google.com",
	}))
	for n, test := range [...]struct {
		Path, Response string
	}{
		{
			Path:     "/",
			Response: optionsPost,
		},
		{
			Path:     "/AAA",
			Response: optionsGetHead,
		},
		{
			Path:     "/BBB",
			Response: optionsPost,
		},
	} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, test.Path, nil))
		if w.Code != http.StatusNoContent {
			t.Errorf("test %d: expecting response code 204, got %d", n+1, w.Code)
		} else if allowed := w.Header().Get("Allow"); allowed != test.Response {
			t.Errorf("test %d: expecting Allow header of %q, got %q", n+1, test.Response, allowed)
		}
	}
}

func TestGet(t *testing.T) {
	f := New(MemStore(map[string]string{
		"AAA": "http://www.google.com",
	}))
	for n, test := range [...]struct {
		Path, Location string
		Status         int
	}{
		{
			Path:     "/",
			Location: "",
			Status:   http.StatusNotFound,
		},
		{
			Path:     "/AAA",
			Location: "http://www.google.com",
			Status:   http.StatusMovedPermanently,
		},
		{
			Path:     "/BBB",
			Location: "",
			Status:   http.StatusNotFound,
		},
	} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, test.Path, nil))
		if w.Code != test.Status {
			t.Errorf("test %d: expecting response code %d, got %d", n+1, test.Status, w.Code)
		} else if url := w.Header().Get("Location"); url != test.Location {
			t.Errorf("test %d: expecting Location header to be %q, got %q", n+1, test.Location, url)
		}
	}
}

type nonrand []int64

func (n *nonrand) Int63() int64 {
	if len(*n) == 0 {
		return 0
	}
	i := (*n)[0]
	*n = (*n)[1:]
	return i
}

func (nonrand) Seed(_ int64) {}

func TestPostBasic(t *testing.T) {
	rs := nonrand{0, 0, 1, 2}
	f := New(MemStore(map[string]string{
		"AA": "http://www.google.com",
	}), RandomSource(&rs), KeyLength(1), URLValidator(HTTPURL))
	for n, test := range [...]struct {
		Body, ContentType, Response string
		Status                      int
	}{
		{
			Body:        "ftp://google.com",
			ContentType: "unknown",
			Response:    unrecognisedContentType,
			Status:      http.StatusUnsupportedMediaType,
		},
		{
			Body:        "ftp://google.com",
			ContentType: "text/plain",
			Response:    invalidURL,
			Status:      http.StatusBadRequest,
		},
		{
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    "AQ",
			Status:      http.StatusOK,
		},
		{
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    "Ag",
			Status:      http.StatusOK,
		},
		{
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    "AAA",
			Status:      http.StatusOK,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.Body))
		r.Header.Set("Content-Type", test.ContentType)
		f.ServeHTTP(w, r)
		if w.Code != test.Status {
			t.Errorf("test %d: expecting response code %d, got %d", n+1, test.Status, w.Code)
		} else if response := strings.TrimSpace(w.Body.String()); response != test.Response {
			t.Errorf("test %d: expecting response %q, got %q", n+1, test.Response, response)
		} else if contentType := w.Header().Get("Content-Type"); w.Code == 200 && contentType != test.ContentType {
			t.Errorf("test %d: expecting return content type %q, got %q", n+1, test.ContentType, contentType)
		}
	}
}
