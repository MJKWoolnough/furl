package furl

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestOptions(t *testing.T) {
	f := New(SetStore(NewStore(Data(map[string]string{
		"AAA": "http://www.google.com",
	}))), KeyValidator(func(key string) bool {
		return key != "ABCD"
	}))
	for n, test := range [...]struct {
		Path, Response string
		Code           int
	}{
		{ // 1
			Path:     "/",
			Code:     http.StatusNoContent,
			Response: optionsPost,
		},
		{ // 2
			Path:     "/AAA",
			Code:     http.StatusNoContent,
			Response: optionsGetHead,
		},
		{ // 3
			Path:     "/BBB",
			Code:     http.StatusNoContent,
			Response: optionsPost,
		},
		{ // 4
			Path:     "/ABCD",
			Code:     http.StatusUnprocessableEntity,
			Response: invalidKey,
		},
	} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest(http.MethodOptions, test.Path, nil))
		if w.Code != test.Code {
			t.Errorf("test %d: expecting response code %d, got %d", n+1, test.Code, w.Code)
		} else if test.Code != http.StatusNoContent {
			if response := strings.TrimSpace(w.Body.String()); response != test.Response {
				t.Errorf("test %d: expecting response %q, got %q", n+1, test.Response, response)
			}
		} else if allowed := w.Header().Get("Allow"); allowed != test.Response {
			t.Errorf("test %d: expecting Allow header of %q, got %q", n+1, test.Response, allowed)
		}
	}
}

func TestGet(t *testing.T) {
	f := New(SetStore(NewStore(Data(map[string]string{
		"AAA": "http://www.google.com",
	}))), KeyValidator(func(key string) bool {
		return key != "ABCD"
	}))
	for n, test := range [...]struct {
		Path, Response string
		Code           int
	}{
		{ // 1
			Path:     "/",
			Code:     http.StatusNotFound,
			Response: "404 page not found",
		},
		{ // 2
			Path:     "/AAA",
			Response: "http://www.google.com",
			Code:     http.StatusMovedPermanently,
		},
		{ // 3
			Path:     "/BBB",
			Code:     http.StatusNotFound,
			Response: "404 page not found",
		},
		{ // 4
			Path:     "/ABCD",
			Code:     http.StatusUnprocessableEntity,
			Response: invalidKey,
		},
	} {
		w := httptest.NewRecorder()
		f.ServeHTTP(w, httptest.NewRequest(http.MethodGet, test.Path, nil))
		if w.Code != test.Code {
			t.Errorf("test %d: expecting response code %d, got %d", n+1, test.Code, w.Code)
		} else if test.Code == http.StatusMovedPermanently {
			if url := w.Header().Get("Location"); url != test.Response {
				t.Errorf("test %d: expecting Location header to be %q, got %q", n+1, test.Response, url)
			}
		} else if response := strings.TrimSpace(w.Body.String()); response != test.Response {
			t.Errorf("test %d: expecting response %q, got %q", n+1, test.Response, response)
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

type postTest struct {
	Body, Key, Response string
	Status              int
}

func testPost(t *testing.T, contentType string, tests []postTest) {
	rs := nonrand{0, 0, 1, 2}
	f := New(SetStore(NewStore(Data(map[string]string{
		"AA": "http://www.google.com",
	}))), RandomSource(&rs), KeyLength(1), URLValidator(HTTPURL), KeyValidator(func(key string) bool {
		return key != "ABC"
	}))
	responseType := contentType
	if contentType == "application/x-www-form-urlencoded" {
		responseType = "text/html"
	}
	for n, test := range tests {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/"+test.Key, strings.NewReader(test.Body))
		r.Header.Set("Content-Type", contentType)
		f.ServeHTTP(w, r)
		if w.Code != test.Status {
			t.Errorf("test %d: expecting response code %d, got %d", n+1, test.Status, w.Code)
		} else if response := strings.TrimSpace(w.Body.String()); response != test.Response {
			t.Errorf("test %d: expecting response %q, got %q", n+1, test.Response, response)
		} else if contentType := w.Header().Get("Content-Type"); w.Code == 200 && contentType != responseType {
			t.Errorf("test %d: expecting return content type %q, got %q", n+1, responseType, contentType)
		}
	}
}

func TestPostText(t *testing.T) {
	testPost(t, "text/plain", []postTest{
		{ // 1
			Body:     "ftp://google.com",
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 2
			Body:     "http://google.com/" + strings.Repeat("A", maxURLLength),
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 3
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 4
			Body:     "http://google.com",
			Key:      "ABC",
			Response: invalidKey,
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 5
			Body:     "http://google.com",
			Key:      "A" + strings.Repeat("A", maxKeyLength),
			Response: invalidKey,
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 6
			Body:     "http://google.com",
			Key:      "AA",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 7
			Body:     "http://google.com",
			Response: "AQ",
			Status:   http.StatusOK,
		},
		{ // 8
			Body:     "http://google.com",
			Key:      "AQ",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 9
			Body:     "http://google.com",
			Key:      "Ag",
			Response: "Ag",
			Status:   http.StatusOK,
		},
		{ // 10
			Body:     "http://google.com",
			Response: "AAA",
			Status:   http.StatusOK,
		},
	})
}

func TestPostJSON(t *testing.T) {
	testPost(t, "application/json", []postTest{
		{ // 1
			Body:     `{"url":a}`,
			Response: fmt.Sprintf(`{"error":%q}`, failedReadRequest),
			Status:   http.StatusBadRequest,
		},
		{ // 2
			Body:     `{"url":"ftp://google.com"}`,
			Response: fmt.Sprintf(`{"error":%q}`, invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 3
			Body:     `{"url":"http://google.com/` + strings.Repeat("A", maxURLLength) + `"}`,
			Response: fmt.Sprintf(`{"error":%q}`, invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 4
			Body:     `{}`,
			Response: fmt.Sprintf(`{"error":%q}`, invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 5
			Body:     `{"url":"http://google.com"}`,
			Key:      "ABC",
			Response: fmt.Sprintf(`{"error":%q}`, invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 6
			Body:     `{"key":"ABC","url":"http://google.com"}`,
			Response: fmt.Sprintf(`{"error":%q}`, invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 7
			Body:     `{"key":"A` + strings.Repeat("A", maxKeyLength) + `","url":"http://google.com"}`,
			Response: fmt.Sprintf(`{"error":%q}`, invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 8
			Body:     `{"url":"http://google.com"}`,
			Key:      "AA",
			Response: fmt.Sprintf(`{"error":%q}`, keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 9
			Body:     `{"key":"AA","url":"http://google.com"}`,
			Response: fmt.Sprintf(`{"error":%q}`, keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 10
			Body:     `{"url":"http://google.com"}`,
			Response: `{"key":"AQ","url":"http://google.com"}`,
			Status:   http.StatusOK,
		},
		{ // 11
			Body:     `{"url":"http://google.com"}`,
			Key:      "AQ",
			Response: fmt.Sprintf(`{"error":%q}`, keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 12
			Body:     `{"key":"AQ","url":"http://google.com"}`,
			Response: fmt.Sprintf(`{"error":%q}`, keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 13
			Body:     `{"url":"http://google.com"}`,
			Key:      "Ag",
			Response: `{"key":"Ag","url":"http://google.com"}`,
			Status:   http.StatusOK,
		},
		{ // 14
			Body:     `{"url":"http://google.com"}`,
			Response: `{"key":"AAA","url":"http://google.com"}`,
			Status:   http.StatusOK,
		},
		{ // 15
			Body:     `{"key":"ABCD","url":"http://google.com"}`,
			Response: `{"key":"ABCD","url":"http://google.com"}`,
			Status:   http.StatusOK,
		},
	})
}

func TestPostXML(t *testing.T) {
	testPost(t, "text/xml", []postTest{
		{ // 1
			Body:     "<furl><url>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", failedReadRequest),
			Status:   http.StatusBadRequest,
		},
		{ // 2
			Body:     "<furl><url>ftp://google.com</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 3
			Body:     "<furl><url>http://google.com/" + strings.Repeat("A", maxURLLength) + "</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 4
			Body:     "<furl></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidURL),
			Status:   http.StatusBadRequest,
		},
		{ // 5
			Body:     "<furl><url>http://google.com</url></furl>",
			Key:      "ABC",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 6
			Body:     "<furl><key>ABC</key><url>http://google.com</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 7
			Body:     "<furl><key>A" + strings.Repeat("A", maxKeyLength) + "</key><url>http://google.com</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", invalidKey),
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 8
			Body:     "<furl><url>http://google.com</url></furl>",
			Key:      "AA",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 9
			Body:     "<furl><key>AA</key><url>http://google.com</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 10
			Body:     "<furl><url>http://google.com</url></furl>",
			Response: "<furl><key>AQ</key><url>http://google.com</url></furl>",
			Status:   http.StatusOK,
		},
		{ // 11
			Body:     "<furl><url>http://google.com</url></furl>",
			Key:      "AQ",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 12
			Body:     "<furl><key>AQ</key><url>http://google.com</url></furl>",
			Response: fmt.Sprintf("<furl><error>%s</error></furl>", keyExists),
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 13
			Body:     "<furl><url>http://google.com</url></furl>",
			Key:      "Ag",
			Response: "<furl><key>Ag</key><url>http://google.com</url></furl>",
			Status:   http.StatusOK,
		},
		{ // 14
			Body:     "<furl><url>http://google.com</url></furl>",
			Response: "<furl><key>AAA</key><url>http://google.com</url></furl>",
			Status:   http.StatusOK,
		},
		{ // 15
			Body:     "<furl><key>ABCD</key><url>http://google.com</url></furl>",
			Response: "<furl><key>ABCD</key><url>http://google.com</url></furl>",
			Status:   http.StatusOK,
		},
	})
}

func TestPostForm(t *testing.T) {
	testPost(t, "application/x-www-form-urlencoded", []postTest{
		{ // 1
			Body:     "url=;",
			Response: failedReadRequest,
			Status:   http.StatusBadRequest,
		},
		{ // 2
			Body:     "url=",
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 3
			Body:     "url=http://google.com/" + strings.Repeat("A", maxURLLength),
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 4
			Response: invalidURL,
			Status:   http.StatusBadRequest,
		},
		{ // 5
			Body:     "url=http://google.com",
			Key:      "ABC",
			Response: invalidKey,
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 6
			Body:     "key=ABC&url=http://google.com",
			Response: invalidKey,
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 7
			Body:     "key=A" + strings.Repeat("A", maxKeyLength) + "&url=http://google.com",
			Response: invalidKey,
			Status:   http.StatusUnprocessableEntity,
		},
		{ // 8
			Body:     "url=http://google.com",
			Key:      "AA",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 9
			Body:     "key=AA&url=http://google.com",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 10
			Body:     "url=http://google.com",
			Response: "AQ",
			Status:   http.StatusOK,
		},
		{ // 11
			Body:     "url=http://google.com",
			Key:      "AQ",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 12
			Body:     "key=AQ&url=http://google.com",
			Response: keyExists,
			Status:   http.StatusMethodNotAllowed,
		},
		{ // 13
			Body:     "url=http://google.com",
			Key:      "Ag",
			Response: "Ag",
			Status:   http.StatusOK,
		},
		{ // 14
			Body:     "url=http://google.com",
			Response: "AAA",
			Status:   http.StatusOK,
		},
		{ // 15
			Body:     "key=ABCD&url=http://google.com",
			Response: "ABCD",
			Status:   http.StatusOK,
		},
	})
}

func TestPostOther(t *testing.T) {
	testPost(t, "unknown", []postTest{
		{ // 1
			Body:     "http://google.com",
			Response: unrecognisedContentType,
			Status:   http.StatusUnsupportedMediaType,
		},
	})
	f := New(CollisionRetries(1), KeyValidator(func(key string) bool {
		return false
	}))
	for n, test := range [...]struct {
		Body, ContentType, Response string
	}{
		{ // 2
			Body:        "http://google.com",
			ContentType: "text/plain",
			Response:    failedKeyGeneration,
		},
		{ // 3
			Body:        `{"url":"http://google.com"}`,
			ContentType: "text/json",
			Response:    fmt.Sprintf(`{"error":%q}`, failedKeyGeneration),
		},
		{ // 4
			Body:        "<furl><url>http://google.com</url></furl>",
			ContentType: "application/xml",
			Response:    fmt.Sprintf("<furl><error>%s</error></furl>", failedKeyGeneration),
		},
		{ // 5
			Body:        "url=http://google.com",
			ContentType: "application/x-www-form-urlencoded",
			Response:    failedKeyGeneration,
		},
	} {
		w := httptest.NewRecorder()
		r := httptest.NewRequest(http.MethodPost, "/", strings.NewReader(test.Body))
		r.Header.Set("Content-Type", test.ContentType)
		f.ServeHTTP(w, r)
		if test.ContentType == "application/x-www-form-urlencoded" {
			test.ContentType = "text/html"
		}
		if w.Code != http.StatusInternalServerError {
			t.Errorf("test %d: expecting response code 500, got %d", n+2, w.Code)
		} else if response := strings.TrimSpace(w.Body.String()); response != test.Response {
			t.Errorf("test %d: expecting response %q, got %q", n+2, test.Response, response)
		} else if contentType := w.Header().Get("Content-Type"); w.Code == 200 && contentType != test.ContentType {
			t.Errorf("test %d: expecting return content type %q, got %q", n+2, test.ContentType, contentType)
		}
	}
}
