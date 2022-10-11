# furl
--
    import "vimagination.zapto.org/furl"

Package Furl provides a drop-in http.Handler that provides short url redirects for longer URLs.

## Usage

#### func  HTTPURL

```go
func HTTPURL(uri string) bool
```
The HTTPURL function can be used with URLValidator to set a simple URL checker that will check for either an http or https scheme, a hostname and no user credentials.

#### type Furl

```go
type Furl struct {
}
```

The Furl type represents a keystore of URLs to either generated or supplied keys.

#### func  New

```go
func New(opts ...Option) *Furl
```
The New function creates a new instance of Furl, with the following defaults that can be changed by adding Option params.
<pre>
urlValidator: By default all strings are treated as valid URLs, this can be changed by using the URLValidator Option.

keyValidator: By default all strings are treated as valid Keys, this can be changed by using the KeyValidator Option.

keyLength:    The default length of generated keys (before base64 encoding) is 6 and can be changed by using the KeyLength Option.

retries:      The default number of retries the key generator will before increasing the key length is 100 and can be changed by using the CollisionRetries Option.

store:        The default store is an empty map that will not permanently record the data. This can be changed by using the SetStore Option.

index:        By default, Furl offers no HTML output. This can be changed by using the Index Option.
</pre>
#### func (*Furl) ServeHTTP

```go
func (f *Furl) ServeHTTP(w http.ResponseWriter, r *http.Request)
```
The ServeHTTP method satifies the http.Handler interface and provides the following endpoints:
<pre>
GET /[key]  - Will redirect the call to the associated URL if it exists, or will return 404 Not Found if it doesn't exists and 422 Unprocessable Entity if the key is invalid.

POST /      - The root can be used to add urls to the store with a generated key. The URL must be specified in the POST body as per the specification below.

POST /[key] - Will attempt to create the specified path with the URL provided as below. If the key is invalid, will respond with 422 Unprocessable Entity. This method cannot be used on existing keys.
</pre>
The URL for the POST methods can be provided in a few content types:

|  Content Type                     |  Example  |
|-----------------------------------|-----------|
| application/json:                 | {"key": "KEY HERE", "url": "URL HERE"} |
| text/xml:                         | <furl><key>KEY HERE</key><url>URL HERE</url></furl> |
| application/x-www-form-urlencoded | key=KEY+HERE&url=URL+HERE |
| text/plain:                       | URL HERE |

For the json, xml, and form content types, the key can be ommitted if it has been supplied in the path or if the key is to be generated.

The response type will be determined by the POST content type:

|  Content Type                     |  Example  |
|-----------------------------------|-----------|
| application/json                  | {"key": "KEY HERE", "url": "URL HERE"} |
| text/xml                          | <furl><key>KEY HERE</key><url>URL HERE</url></furl> |
| text/plain                        | KEY HERE |

For application/x-www-form-urlencoded, the content type of the return will be text/html and the response will match that of text/plain.

#### type Option

```go
type Option func(*Furl)
```

The Option type is used to specify optional params to the New function call

#### func  CollisionRetries

```go
func CollisionRetries(retries uint) Option
```
The CollisionRetries Option sets how many tries a Furl instance will retry generating keys at a given length before increasing the length in order to find a unique key.

#### func  Index

```go
func Index(index func(w http.ResponseWriter, r *http.Request, code int, output string)) Option
```
The Index Option allows for custom error and success output.

For a POST request with codee http.StatusOK (200), the output will be the generated or specified key. In all other times, the output is the error string corresponding to the error code.

NB: The index function won't be called for JSON, XML, or Text POST requests.

#### func  KeyLength

```go
func KeyLength(length uint) Option
```
The KeyLength Option sets the minimum key length on a Furl instance.

NB: The key length is the length of the generated key before base64 encoding, which will increase the size. The actual key length will be the result of base64.RawURLEncoding.EncodedLen(length).

#### func  KeyValidator

```go
func KeyValidator(fn func(key string) bool) Option
```
The KeyValidator Option allows a Furl instance to validate both generated and suggested keys against a set of custom criteria.

If the passed function returns false the Key passed to it will be considered invalid and will either generate a new one, if it was generated to begin with, or simply reject the suggested key.

#### func  RandomSource

```go
func RandomSource(source rand.Source) Option
```
The RandomSource Option allows the specifying of a custom source of randomness.

#### func  SetStore

```go
func SetStore(s Store) Option
```
The SetStore options allows for setting both starting data and the options to persist the collected data. See the Store interface and NewStore function for more information about Stores.

#### func  URLValidator

```go
func URLValidator(fn func(url string) bool) Option
```
The URLValidator Option allows a Furl instance to validate URLs against a custom set of criteria.

If the passed function returns false the URL passed to it will be considered invalid and will not be stored and not be assigned a key.

#### type Store

```go
type Store interface {
	Get(key string) (string, bool)
	Tx(func(tx Tx))
}
```

The Store interface allows for setting a custom storage solution to Furl, such as a database or keystore.

The Get method should be used to retireve the URL associated with the passed key.

The Tx method should start a thread safe writing context that will be used for creating new keys. See the Tx interface for more details.

#### func  NewStore

```go
func NewStore(opts ...StoreOption) Store
```
NewStore creates a map based implementation of the Store interface, with the following defaults that can be changed by adding StoreOption params:

urls: By default, the Store is created with an empty map. This can be changed with the Data StoreOption.

save: By default, there is no permanent storage of the key:url map. This can be changed by the Save StoreOption.

#### type StoreOption

```go
type StoreOption func(*mapStore)
```

The StoreOption type is used to specify optional params to the NewStore function
call.

#### func  Data

```go
func Data(data map[string]string) StoreOption
```
The Data StoreOption is used to set the initial map of keys -> urls. The passed map should not be accessed by anything other than Furl until Furl is no longer is use.

NB: Neither the keys or URLs are checked to be valid.

#### func  Save

```go
func Save(save func(key, url string)) StoreOption
```
The Save StoreOption is used to set a function that stores the keys and urls outside of Furl. For example, could be used to write to a file that be later loaded to provide the data for a future instance of Furl.

#### type Tx

```go
type Tx interface {
	Has(key string) bool
	Set(key, url string)
}
```

The Tx interface represents a thread safe writing context for generating and storing keys and their corresponding URLs.

The Has method may be called multiple times per Store.Tx call.

The Set method will be called at most one time per Store.Tx call, and will be used to set the uniquely generated or passed key and its corresponding URL. The implementation of this method can be used to provide a more permanent storage for the key:url store.
