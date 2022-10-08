package furl

import (
	"io"
	"math/rand"
	"net/url"

	"vimagination.zapto.org/byteio"
)

type Option func(*Furl)

func URLValidator(fn func(string) bool) Option {
	return func(f *Furl) {
		f.urlValidator = fn
	}
}

func HTTPURL(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

func KeyValidator(fn func(string) bool) Option {
	return func(f *Furl) {
		f.keyValidator = fn
	}
}

func KeyLength(length uint) Option {
	return func(f *Furl) {
		f.keyLength = length
	}
}

func CollisionRetries(retries uint) Option {
	return func(f *Furl) {
		f.retries = retries
	}
}

func MemStore(urls map[string]string) Option {
	return func(f *Furl) {
		f.urls = urls
	}
}

func IOStore(rw io.ReadWriter) Option {
	return func(f *Furl) {
		f.urls = make(map[string]string)
		r := byteio.StickyLittleEndianReader{Reader: rw}
		for {
			key := r.ReadString16()
			if key == "" {
				break
			}
			f.urls[key] = r.ReadString16()
		}
		if r.Err != nil && r.Err != io.EOF {
			panic(r.Err)
		}
		w := byteio.StickyLittleEndianWriter{Writer: rw}
		f.save = func(key string, url string) error {
			w.WriteString16(key)
			w.WriteString16(url)
			if w.Err == nil {
				if f, ok := rw.(interface{ Sync() error }); ok {
					w.Err = f.Sync()
				}
			}
			return w.Err
		}
	}
}

func RandomSource(source rand.Source) Option {
	return func(f *Furl) {
		f.rand = rand.New(source)
	}
}
