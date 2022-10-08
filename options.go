package furl

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
