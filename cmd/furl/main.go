package main

import (
	"bufio"
	"bytes"
	"context"
	_ "embed"
	"flag"
	"fmt"
	"html/template"
	"io"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"vimagination.zapto.org/furl"
)

//go:embed index.tmpl
var index string

func main() {
	if err := run(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func keyValidator(key string) bool {
	for _, c := range key {
		if !strings.ContainsRune("ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_", c) {
			return false
		}
	}
	return true
}

type wrappedResponseWriter struct {
	http.ResponseWriter
	code int
	bytes.Buffer
}

func (w *wrappedResponseWriter) WriteHeader(code int) {
	w.code = code
}

func (w *wrappedResponseWriter) Write(p []byte) (int, error) {
	switch w.code {
	case http.StatusOK:
		return w.Buffer.Write(p)
	case http.StatusBadRequest, http.StatusUnprocessableEntity, http.StatusMethodNotAllowed:
		return len(p), nil
	default:
		if w.code != 0 {
			w.ResponseWriter.WriteHeader(w.code)
			w.code = 0
		}
		return w.ResponseWriter.Write(p)
	}
}

type tmplVars struct {
	Success, URL, URLError, Key, KeyError string
}

func run() error {
	tmpl := template.Must(template.New("").Parse(index))
	file := flag.String("f", "", "filename to store key:url map data")
	port := flag.Int("p", 8080, "port for server to listen on")
	serverURL := flag.String("s", "", "base server url. e.g. http://furl.com/")
	flag.Parse()

	furlParams := []furl.Option{
		furl.URLValidator(furl.HTTPURL),
		furl.KeyValidator(keyValidator),
	}

	if *file != "" {
		f, err := os.OpenFile(*file, os.O_RDWR|os.O_CREATE, 0o666)
		if err != nil {
			return fmt.Errorf("error opening database file (%s): %w", *file, err)
		}
		defer f.Close()
		data := make(map[string]string)
		b := bufio.NewReader(f)
		var length [2]byte
		for {
			if _, err := io.ReadFull(b, length[:]); err != nil {
				return fmt.Errorf("error reading key length: %w", err)
			}
			keyLength := int(length[0]) | (int(length[1]) << 8)
			if keyLength == 0 {
				break
			}
			key := make([]byte, keyLength)
			if _, err = io.ReadFull(b, key); err != nil {
				return fmt.Errorf("error reading key: %w", err)
			}
			if urlLength := int(length[0]) | (int(length[1]) << 8); urlLength > 0 {
				if _, err := io.ReadFull(b, length[:]); err != nil {
					return fmt.Errorf("error reading url length: %w", err)
				}
				url := make([]byte, urlLength)
				if _, err = io.ReadFull(b, url); err != nil {
					return fmt.Errorf("error reading url: %w", err)
				}
				data[string(key)] = string(url)
			}
		}
		w := bufio.NewWriter(f)
		furlParams = append(furlParams, furl.SetStore(furl.NewStore(furl.Data(data), furl.Save(func(key, url string) {
			length[0] = byte(len(key))
			length[1] = byte(len(key) >> 8)
			if _, err := w.Write(length[:]); err != nil {
				panic(fmt.Errorf("error while writing key length: %w", err))
			}
			if _, err := w.WriteString(key); err != nil {
				panic(fmt.Errorf("error while writing key: %w", err))
			}
			length[0] = byte(len(url))
			length[1] = byte(len(url) >> 8)
			if _, err := w.Write(length[:]); err != nil {
				panic(fmt.Errorf("error while writing url length: %w", err))
			}
			if _, err := w.WriteString(url); err != nil {
				panic(fmt.Errorf("error while writing url: %w", err))
			}
			if err := w.Flush(); err != nil {
				panic(fmt.Errorf("error while flushing buffers: %w", err))
			}
			if err := f.Sync(); err != nil {
				panic(fmt.Errorf("error while syncing file: %w", err))
			}
		}))))
	}
	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *port})
	if err != nil {
		return fmt.Errorf("error listening on port %d: %w", *port, err)
	}

	f := furl.New(furlParams...)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/" {
				tmpl.Execute(w, tmplVars{})
			} else if r.Method == http.MethodPost && r.URL.Path == "/" && r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
				wr := wrappedResponseWriter{
					ResponseWriter: w,
					code:           http.StatusOK,
				}
				f.ServeHTTP(&wr, r)
				tv := tmplVars{
					Key: r.PostForm.Get("key"),
					URL: r.PostForm.Get("url"),
				}
				switch wr.code {
				case http.StatusOK:
					tv.Success = *serverURL + wr.Buffer.String()
				case http.StatusBadRequest:
					tv.URLError = "Invalid URL"
				case http.StatusUnprocessableEntity:
					tv.KeyError = "Invalid Alias"
				case http.StatusMethodNotAllowed:
					tv.KeyError = "Alias Exists"
				default:
					return
				}
				tmpl.Execute(w, tv)
			} else {
				f.ServeHTTP(w, r)
			}
		}),
	}

	go server.Serve(l)

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	signal.Stop(sc)
	close(sc)

	return server.Shutdown(context.Background())
}
