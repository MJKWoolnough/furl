package furl

import (
	"io"

	"vimagination.zapto.org/byteio"
)

type Option func(*Furl)

func URLValidator(fn func(string) bool) Option {
	return func(f *Furl) {
		f.urlValidator = fn
	}
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
			key := r.ReadStringX()
			if key == "" {
				break
			}
			f.urls[key] = r.ReadStringX()
		}
		if r.Err != nil && r.Err != io.EOF {
			panic(r.Err)
		}
		w := byteio.StickyLittleEndianWriter{Writer: rw}
		f.save = func(key string, url string) error {
			w.WriteStringX(key)
			w.WriteStringX(url)
			if w.Err == nil {
				if f, ok := rw.(interface{ Sync() error }); ok {
					w.Err = f.Sync()
				}
			}
			return w.Err
		}
	}
}
