package furl

import (
	"net/http"
	"net/http/httptest"
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
