package wok

import (
	"context"
	"io"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/manvalls/way"
)

type headerAndValue struct {
	header string
	value  string
}

// Request wraps an HTTP request
type Request struct {
	*http.Request
	http.ResponseWriter
	context.Context
	way.Router
	url.Values

	*StatusCodeGetterSetter

	RequestHeader  http.Header
	ResponseHeader http.Header

	route []uint
	index int

	routes      map[*struct{}]headerAndValue
	routesMutex *sync.Mutex

	vary      map[string]int
	varyMutex *sync.Mutex

	customBody       *bool
	customBodyReader *io.Reader
	customBodyMutex  *sync.Mutex

	redirectedRoute  *[]uint
	redirectedParams *Params
	redirectCond     *sync.Cond
	fullParams       Params
}

// UseEmptyBody tells the handler to send an empty body as the response of this request
func (r Request) UseEmptyBody() {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	r.customBodyMutex.Lock()
	defer r.customBodyMutex.Unlock()

	*r.customBody = true
	*r.customBodyReader = nil
	r.redirectCond.Broadcast()
}

// UseCustomBody tells the handler to send the provided reader as the response of this request
func (r Request) UseCustomBody(reader io.Reader) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	r.customBodyMutex.Lock()
	defer r.customBodyMutex.Unlock()

	*r.customBody = true
	*r.customBodyReader = reader
	r.redirectCond.Broadcast()
}

// ContextVary adds several headers to the Vary header, for the
// duration of this context
func (r Request) ContextVary(headers ...string) {
	r.varyMutex.Lock()
	defer r.varyMutex.Unlock()

	for _, header := range headers {
		r.vary[header]++
	}

	go func() {
		<-r.Done()
		r.varyMutex.Lock()
		defer r.varyMutex.Unlock()

		for _, header := range headers {
			if r.vary[header] == 1 {
				delete(r.vary, header)
			} else {
				r.vary[header]--
			}
		}
	}()
}

// Vary adds several headers to the Vary header
func (r Request) Vary(headers ...string) {
	r.varyMutex.Lock()
	defer r.varyMutex.Unlock()

	for _, header := range headers {
		r.vary[header]++
	}
}

// Params hold the list of request parameters
type Params = map[string][]string

// FromHeader builds a route path from an HTTP header
func (r Request) FromHeader(header string) (Params, []uint) {
	header = http.CanonicalHeaderKey(header)
	headerValue := strings.Join(r.Request.Header[header], ",")

	rawRoute := ""
	rawQuery := ""

	parts := strings.Split(headerValue, "?")

	switch len(parts) {
	case 0:
	case 1:
		rawRoute = parts[0]
	default:
		rawRoute = parts[0]
		rawQuery = parts[1]
	}

	route := make([]uint, 0)
	for _, h := range strings.Split(rawRoute, ",") {
		n, err := strconv.ParseUint(strings.Trim(h, " "), 36, 64)
		if err == nil {
			route = append(route, uint(n))
		}
	}

	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		query = url.Values{}
	}

	return query, route
}

// ToHeader maps a route path to a header value
func ToHeader(params Params, route ...uint) string {
	result := make([]string, len(route))
	for i, v := range route {
		result[i] = strconv.FormatUint(uint64(v), 36)
	}

	var values url.Values
	path := strings.Join(result, ",")
	values = params
	query := values.Encode()

	if query != "" {
		return path + "?" + query
	}

	return path
}

// StatusCodeGetterSetter holds a status code in a concurrent-safe way
type StatusCodeGetterSetter struct {
	mutex      sync.Mutex
	statusCode int
}

// StatusCode retrieves the internal status code
func (sc *StatusCodeGetterSetter) StatusCode() int {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()

	if sc.statusCode == 0 {
		return 200
	}

	return sc.statusCode
}

// SetStatusCode sets the internal status code
func (sc *StatusCodeGetterSetter) SetStatusCode(statusCode int) {
	sc.mutex.Lock()
	defer sc.mutex.Unlock()
	sc.statusCode = statusCode
}

// - URL redirections

// URLRedirect issues an HTTP redirection
func (r Request) URLRedirect(statusCode int, params way.Params, route ...uint) error {
	redirURL, err := r.GetURL(params, route...)
	if err != nil {
		return err
	}

	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	r.customBodyMutex.Lock()
	defer r.customBodyMutex.Unlock()

	*r.customBody = true
	*r.customBodyReader = nil

	r.ResponseHeader.Set("Location", redirURL)
	r.SetStatusCode(statusCode)
	r.redirectCond.Broadcast()
	return nil
}

// PartialURLRedirect issues an HTTP redirection starting from the current route level
func (r Request) PartialURLRedirect(statusCode int, params way.Params, route ...uint) error {
	return r.URLRedirect(statusCode, params, append(way.Clone(r.route[:r.index]), route...)...)
}

// ParamsURLRedirect issues an HTTP redirection changing only route parameters
func (r Request) ParamsURLRedirect(statusCode int, params way.Params) error {
	return r.URLRedirect(statusCode, params, r.route...)
}

// - Internal redirections

// Redirect issues an internal redirection at the current handler
func (r Request) Redirect(params Params, route ...uint) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	*r.redirectedRoute = route
	*r.redirectedParams = params
	r.redirectCond.Broadcast()
}

// PartialRedirect issues an internal redirection at the current handler,
// starting from the current route level
func (r Request) PartialRedirect(params Params, route ...uint) {
	r.Redirect(params, append(way.Clone(r.route[:r.index]), route...)...)
}

// ParamsRedirect issues an internal redirection changing only route parameters
func (r Request) ParamsRedirect(params Params) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	*r.redirectedParams = params
	r.redirectCond.Broadcast()
}

// ChangeParams applies a modifier function to the full parameters of this request
func (r Request) ChangeParams(modifier func(Params)) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	newParams := cloneParams(r.fullParams)
	modifier(newParams)

	*r.redirectedParams = newParams
	r.redirectCond.Broadcast()
}

// - Aliases

// MaxBytesReader limits the size of a reader
func (r Request) MaxBytesReader(rc io.ReadCloser, n int64) io.ReadCloser {
	return http.MaxBytesReader(r.ResponseWriter, rc, n)
}

// ServeContent replies to the request using the content in the provided ReadSeeker
func (r Request) ServeContent(name string, modtime time.Time, content io.ReadSeeker) {
	http.ServeContent(r.ResponseWriter, r.Request, name, modtime, content)
}

// ServeFile replies to the request with the contents of the named file or directory
func (r Request) ServeFile(name string) {
	http.ServeFile(r.ResponseWriter, r.Request, name)
}

// SetCookie adds a Set-Cookie header to the provided ResponseWriter's headers
func (r Request) SetCookie(cookie *http.Cookie) {
	http.SetCookie(r.ResponseWriter, cookie)
}
