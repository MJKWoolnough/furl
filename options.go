package furl

import (
	"io"
	"math/rand"
	"net/url"

	"vimagination.zapto.org/byteio"
)

// The Option type is used to specify optional params to the New function call
type Option func(*Furl)

// The URLValidator Option allows a Furl instance to validate URLs against a
// custom set of criteria.
//
// If the passed function returns false the URL passed to it will be considered
// invalid and will not be stored and not be assigned a key.
func URLValidator(fn func(url string) bool) Option {
	return func(f *Furl) {
		f.urlValidator = fn
	}
}

// The HTTPURL function can be used with URLValidator to set a simple URL
// checker that will check for either an http or https scheme, a hostname and
// no user credentials.
func HTTPURL(uri string) bool {
	u, err := url.Parse(uri)
	if err != nil {
		return false
	}
	return (u.Scheme == "http" || u.Scheme == "https") && u.Hostname() != "" && u.User == nil
}

// The KeyValidator Option allows a Furl instance to validate both generated
// and suggested keys against a set of custom criteria.
//
// If the passed function returns false the Key passed to it will be considered
// invalid and will either generate a new one, if it was generated to begin
// with, or simply reject the suggested key.
func KeyValidator(fn func(key string) bool) Option {
	return func(f *Furl) {
		f.keyValidator = fn
	}
}

// The KeyLength Option sets the minimum key length on a Furl instance.
//
// NB: The key length is the length of the generated key before base64
// encoding, which will increase the size. The actual key length will be
// the result of base64.RawURLEncoding.EncodedLen(length).
func KeyLength(length uint) Option {
	return func(f *Furl) {
		f.keyLength = length
	}
}

// The CollisionRetries Option sets how many tries a Furl instance will retry
// generating keys at a given length before increasing the length in order to
// find a unique key.
func CollisionRetries(retries uint) Option {
	return func(f *Furl) {
		f.retries = retries
	}
}

// The MemStore option allows setting a custom filled map of keys -> urls. The
// passed map should not be accessed by anything other than Furl until Furl is
// no longer is use.
//
// NB: Neither the keys or URLs are checked to be valid.
func MemStore(urls map[string]string) Option {
	return func(f *Furl) {
		f.urls = urls
	}
}

// The IOStore sets io.ReadWriter to load and save keys and URLs to.
//
// Each key:url pair is stored sequentially and according to the following
// format:
// struct {
//	KeyLength uint16
//      Key       [KeyLength]byte
//      URLLength uint16
//      URL       [URLLength]byte
// }
//
// NB: The uint16s are store in LittleEndian format.
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

// The RandomSource Option allows the specifying of a custom source of
// randomness.
func RandomSource(source rand.Source) Option {
	return func(f *Furl) {
		f.rand = rand.New(source)
	}
}
