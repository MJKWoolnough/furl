package furl

import (
	"bytes"
	"testing"
)

func TestIOStore(t *testing.T) {
	const (
		Key1          = "ABCDE"
		Key2          = "654321"
		Key3          = "NEWKEY"
		Key4          = "NewKey"
		URL1          = "https://www.example.com"
		URL2          = "http://www.google.com"
		URL3          = "http://microsoft.com"
		URL4          = "http://yahoo.com"
		WrittenString = "\x06\x00NEWKEY\x14\x00http://microsoft.com\x06\x00NewKey\x10\x00http://yahoo.com"
	)
	var (
		f Furl
		b bytes.Buffer
	)
	b.WriteByte(byte(len(Key1)))
	b.WriteByte(0)
	b.WriteString(Key1)
	b.WriteByte(byte(len(URL1)))
	b.WriteByte(0)
	b.WriteString(URL1)
	b.WriteByte(byte(len(Key2)))
	b.WriteByte(0)
	b.WriteString(Key2)
	b.WriteByte(byte(len(URL2)))
	b.WriteByte(0)
	b.WriteString(URL2)
	IOStore(&b)(&f)
	if len(f.urls) != 2 {
		t.Errorf("test 1: expecting 2 entries in URL map, got %d", len(f.urls))
	}
	if url, ok := f.urls[Key1]; !ok {
		t.Errorf("test 2: with key %q, expecting to get URL %q, got no url", Key1, URL1)
	} else if url != URL1 {
		t.Errorf("test 2: with key %q, expecting to get URL %q, got %q", Key1, URL1, url)
	}
	if url, ok := f.urls[Key2]; !ok {
		t.Errorf("test 3: with key %q, expecting to get URL %q, got no url", Key2, URL2)
	} else if url != URL2 {
		t.Errorf("test 3: with key %q, expecting to get URL %q, got %q", Key2, URL2, url)
	}
	f.save(Key3, URL3)
	f.save(Key4, URL4)
	if str := b.String(); str != WrittenString {
		t.Errorf("test 4: failed to correctly write new key/url, expecting %q, got %q", WrittenString, str)
	}
}
