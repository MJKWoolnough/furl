package main

import (
	"bufio"
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
	"path"
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

type tmplVars struct {
	Success, URL, URLError, Key, KeyError string
	NotFound                              bool
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
		furl.Index(func(w http.ResponseWriter, r *http.Request, code int, data string) {
			if r.Method == http.MethodGet {
				isRoot := r.URL.Path == "/" || r.URL.Path == ""
				if code == http.StatusUnprocessableEntity && !isRoot {
					http.Redirect(w, r, "/", http.StatusFound)
					return
				}
				var tv tmplVars
				if code == http.StatusNotFound && !isRoot {
					w.WriteHeader(code)
					tv.NotFound = true
					tv.Key = path.Base("/" + r.URL.Path)
				}
				tmpl.Execute(w, tv)
			} else if r.Method == http.MethodPost {
				tv := tmplVars{
					URL: r.PostForm.Get("url"),
				}
				switch code {
				case http.StatusOK:
					tv.Key = data
					tv.Success = *serverURL + data
				case http.StatusBadRequest:
					tv.URLError = "Invalid URL"
				case http.StatusUnprocessableEntity:
					tv.KeyError = "Invalid Alias"
				case http.StatusMethodNotAllowed:
					tv.KeyError = "Alias Exists"
				}
				tmpl.Execute(w, tv)
			}
		}),
	}

	if *file != "" { // if we're loading a file-back store
		f, err := os.OpenFile(*file, os.O_RDWR|os.O_CREATE, 0o666)
		if err != nil {
			return fmt.Errorf("error opening database file (%s): %w", *file, err)
		}
		defer f.Close()
		data := make(map[string]string)
		b := bufio.NewReader(f)
		var length [2]byte

		/*
			Each key:url pair is stored sequentially and according to the following format:

			struct {
				KeyLength uint16
				Key       [KeyLength]byte
				URLLength uint16
				URL       [URLLength]byte
			}

			The uint16s are store in LittleEndian format.
		*/

		for {
			if _, err := io.ReadFull(b, length[:]); err != nil && err != io.EOF {
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
			if _, err := io.ReadFull(b, length[:]); err != nil && err != io.EOF {
				return fmt.Errorf("error reading url length: %w", err)
			}
			if urlLength := int(length[0]) | (int(length[1]) << 8); urlLength > 0 {
				url := make([]byte, urlLength)
				if _, err = io.ReadFull(b, url); err != nil {
					return fmt.Errorf("error reading url: %w", err)
				}
				data[string(key)] = string(url)
			}
			length[0] = 0
			length[1] = 0
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

	server := &http.Server{
		Handler: furl.New(furlParams...),
	}

	go server.Serve(l)

	// wait for SIGINT

	sc := make(chan os.Signal, 1)
	signal.Notify(sc, os.Interrupt)
	<-sc
	signal.Stop(sc)
	close(sc)

	return server.Shutdown(context.Background())
}
