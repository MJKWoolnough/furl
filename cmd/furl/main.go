package main

import (
	"bytes"
	"context"
	"flag"
	"fmt"
	"net"
	"net/http"
	"os"
	"os/signal"
	"strings"

	"vimagination.zapto.org/furl"
)

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
		return w.ResponseWriter.Write(p)
	}
}

func run() error {
	file := flag.String("f", "", "filename to store key:url map data")
	port := flag.Int("p", 8080, "port for server to listen on")
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
		furlParams = append(furlParams, furl.IOStore(f))
	}
	l, err := net.ListenTCP("tcp", &net.TCPAddr{Port: *port})
	if err != nil {
		return fmt.Errorf("error listening on port %d: %w", *port, err)
	}

	f := furl.New(furlParams...)

	server := &http.Server{
		Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == http.MethodGet && r.URL.Path == "/" {
				// serve index
			} else if r.Method == http.MethodPost && r.URL.Path == "/" && r.Header.Get("Content-Type") == "application/x-www-form-urlencoded" {
				wr := wrappedResponseWriter{
					ResponseWriter: w,
					code:           http.StatusOK,
				}
				f.ServeHTTP(&wr, r)
				switch wr.code {
				case http.StatusOK:
					// generate page with new key
				case http.StatusBadRequest:
					// generate page with URL invalid error
				case http.StatusUnprocessableEntity:
					// generate page with Key invalid error
				case http.StatusMethodNotAllowed:
					// generate page with Key exists error
				}
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
