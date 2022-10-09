package furl

import (
	"math/rand"
	"net/url"
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

func SetStore(s Store) Option {
	return func(f *Furl) {
		f.store = s
	}
}

// The RandomSource Option allows the specifying of a custom source of
// randomness.
func RandomSource(source rand.Source) Option {
	return func(f *Furl) {
		f.rand = rand.New(source)
	}
}
