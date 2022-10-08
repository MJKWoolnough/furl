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

func TestPost(t *testing.T) {
	rs := nonrand{0, 0, 1, 2}
	f := New(MemStore(map[string]string{
		"AA": "http://www.google.com",
	}), RandomSource(&rs), KeyLength(1), URLValidator(HTTPURL), KeyValidator(func(key string) bool {
		return key != "ABC"
	}))
	for n, test := range [...]struct {
		Body, Key, ContentType, Response string
		Status                           int
	}{
		{ // 1
			Body:        "ftp://google.com",
			ContentType: "unknown",
			Response:    unrecognisedContentType,
			Status:      http.StatusUnsupportedMediaType,
		},
		{ // 2
			Body:        "ftp://google.com",
			ContentType: "text/plain",
			Response:    invalidURL,
			Status:      http.StatusBadRequest,
		},
		{ // 3
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    "AQ",
			Status:      http.StatusOK,
		},
		{ // 4
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    "Ag",
			Status:      http.StatusOK,
		},
		{ // 5
			Body:        "http://google.com",
			Key:         "ABC",
			ContentType: "text/plain",
			Response:    invalidKey,
			Status:      http.StatusUnprocessableEntity,
		},
		{ // 6
			Body:        `{"url":"http://google.com"}`,
			Key:         "ABC",
			ContentType: "application/json",
			Response:    `{"error":"` + invalidKey + "\"}",
			Status:      http.StatusUnprocessableEntity,
		},
		{ // 7
			Body:        `<furl><url>http://google.com</url></furl>`,
			Key:         "ABC",
			ContentType: "text/xml",
			Response:    "<furl><error>" + invalidKey + "</error></furl>",
			Status:      http.StatusUnprocessableEntity,
		},
		{ // 8
			Body:        "url=http://google.com",
			Key:         "ABC",
			ContentType: "application/x-www-form-urlencoded",
			Response:    invalidKey,
			Status:      http.StatusUnprocessableEntity,
		},
		{ // 9
			Body:        `{"url":"http://google.com"}`,
			Key:         "ABC",
			ContentType: "application/json",
			Response:    `{"error":"` + invalidKey + "\"}",
			Status:      http.StatusUnprocessableEntity,
		},
		{ // 10
			Body:        "<furl><url>http://google.com</url></furl>",
			ContentType: "text/xml",
			Response:    "<furl><key>AAA</key><url>http://google.com</url></furl>",
			Status:      http.StatusOK,
		},
		{ // 11
			Body:        `{"url":"http://api.microsoft.com"}`,
			ContentType: "application/json",
			Response:    `{"key":"AAAA","url":"http://api.microsoft.com"}`,
			Status:      http.StatusOK,
		},
		{ // 12
			Body:        "url=https://www.example.com",
			ContentType: "application/x-www-form-urlencoded",
			Response:    "AAAAAA",
			Status:      http.StatusOK,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/"+test.Key, strings.NewReader(test.Body))
		r.Header.Set("Content-Type", test.ContentType)
		if test.ContentType == "application/x-www-form-urlencoded" {
			test.ContentType = "text/html"
		}
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
