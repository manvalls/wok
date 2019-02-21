package wok

import (
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/manvalls/way"
	"github.com/manvalls/wit"
)

type headerAndValue struct {
	header string
	value  string
}

type deduper struct {
	loadedDependencies     map[string]bool
	firstDeduperElement    *deduperElement
	lastDeduperElement     *deduperElement
	indexedDeduperElements map[string]*deduperElement
	deduperMutex           *sync.Mutex
}

// ReadOnlyRequest wraps a read-only HTTP request
type ReadOnlyRequest struct {
	*http.Request
	context.Context
	*sync.Mutex
	way.Router
	url.Values
	InstanceID    string
	OldParams     url.Values
	IsNavigation  bool
	IsSocket      bool
	InitialLoad   bool
	Call          CallData
	Input         <-chan url.Values
	Output        chan<- wit.Command
	RequestHeader http.Header
}

var errClosed = errors.New("Socket closed")

// Send sends an command through a socket, or returns an error if the socket is closed
func (r ReadOnlyRequest) Send(command wit.Command) error {
	select {
	case r.Output <- command:
		return nil
	case <-r.Done():
		return errClosed
	}
}

// CmdCh returns Output, useful to meet interface definitions
func (r ReadOnlyRequest) CmdCh() chan<- wit.Command {
	return r.Output
}

// ClientID returns InstanceID, useful to meet interface definitions
func (r ReadOnlyRequest) ClientID() string {
	return r.InstanceID
}

// Request wraps an HTTP request
type Request struct {
	ReadOnlyRequest
	w http.ResponseWriter
	*StatusCodeGetterSetter
	ResponseHeader http.Header

	route []string
	index int

	routes      map[*struct{}]headerAndValue
	routesMutex *sync.Mutex

	vary      map[string]int
	varyMutex *sync.Mutex

	custom        *bool
	customHandler *func(http.ResponseWriter)
	customMutex   *sync.Mutex

	redirectedRoute  *[]string
	redirectedParams *Params
	redirectCond     *sync.Cond
	fullParams       Params

	*deduper
}

// HandleResponse tells the controller how to handle the response
func (r Request) HandleResponse(f func(http.ResponseWriter)) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	r.customMutex.Lock()
	defer r.customMutex.Unlock()

	if *r.custom {
		return
	}

	*r.custom = true
	*r.customHandler = f
	r.redirectCond.Broadcast()
}

// HandleBody tells the controller how to handle the response body
func (r Request) HandleBody(f func(http.ResponseWriter)) {
	r.HandleResponse(func(w http.ResponseWriter) {
		code := r.StatusCode()
		if code != http.StatusOK {
			w.WriteHeader(code)
		}

		f(w)
	})
}

// UseEmptyBody tells the controller to send an empty body as the response of this request
func (r Request) UseEmptyBody() {
	r.HandleBody(nil)
}

// UseCustomBody tells the controller to send the provided reader as the response of this request
func (r Request) UseCustomBody(reader io.Reader) {
	r.HandleBody(func(w http.ResponseWriter) {
		io.Copy(w, reader)
	})
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
func (r ReadOnlyRequest) FromHeader(header string) (Params, []string) {
	header = http.CanonicalHeaderKey(header)

	h, ok := r.Request.Header[header]
	if !ok {
		return url.Values{}, []string{}
	}

	headerValue := strings.Join(h, ",")

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

	route := make([]string, 0)
	for _, h := range strings.Split(rawRoute, ",") {
		route = append(route, strings.Trim(h, " "))
	}

	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		query = url.Values{}
	}

	return query, route
}

// ToHeader maps a route path to a header value
func ToHeader(params Params, route ...string) string {
	result := make([]string, len(route))
	for i, v := range route {
		result[i] = v
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
func (r Request) URLRedirect(statusCode int, params way.Params, route ...string) error {
	redirURL, err := r.GetURL(params, route...)
	if err != nil {
		return err
	}

	r.HandleResponse(func(w http.ResponseWriter) {
		w.Header().Set("Location", redirURL)
		w.WriteHeader(statusCode)
	})

	return nil
}

// PartialURLRedirect issues an HTTP redirection starting from the current route level
func (r Request) PartialURLRedirect(statusCode int, params way.Params, route ...string) error {
	return r.URLRedirect(statusCode, params, append(way.Clone(r.route[1:r.index]), route...)...)
}

// ParamsURLRedirect issues an HTTP redirection changing only route parameters
func (r Request) ParamsURLRedirect(statusCode int, params way.Params) error {
	return r.URLRedirect(statusCode, params, r.route...)
}

// - Internal redirections

func (r Request) redirect(params Params, route ...string) {
	r.redirectCond.L.Lock()
	defer r.redirectCond.L.Unlock()

	*r.redirectedRoute = route
	*r.redirectedParams = params
	r.redirectCond.Broadcast()
}

// Redirect issues an internal redirection at the current controller
func (r Request) Redirect(params Params, route ...string) {
	r.redirect(params, append([]string{r.route[0]}, route...)...)
}

// PartialRedirect issues an internal redirection at the current controller,
// starting from the current route level
func (r Request) PartialRedirect(params Params, route ...string) {
	r.redirect(params, append(way.Clone(r.route[:r.index]), route...)...)
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

// - Deduper

type deduperElement struct {
	next *deduperElement
	prev *deduperElement
	key  string
	n    uint
}

// Load marks the provided dependencies as required
func (r Request) Load(dependencies ...string) {
	r.deduperMutex.Lock()
	defer r.deduperMutex.Unlock()

	for _, depency := range dependencies {
		r.load(depency)
	}
}

func (r Request) load(dependency string) {
	if r.loadedDependencies[dependency] {
		return
	}

	elem := r.indexedDeduperElements[dependency]
	if elem == nil {
		elem = &deduperElement{
			prev: r.lastDeduperElement,
			key:  dependency,
		}

		r.lastDeduperElement = elem
		if r.firstDeduperElement == nil {
			r.firstDeduperElement = elem
		}
	}

	elem.n++
	go func() {
		<-r.Done()
		r.deduperMutex.Lock()
		defer r.deduperMutex.Unlock()

		elem.n--
		if elem.n > 0 {
			return
		}

		if r.firstDeduperElement == elem {
			r.firstDeduperElement = elem.next
		}

		if r.lastDeduperElement == elem {
			r.lastDeduperElement = elem.prev
		}

		if elem.prev != nil {
			elem.prev.next = elem.next
		}

		if elem.next != nil {
			elem.next.prev = elem.prev
		}

		delete(r.indexedDeduperElements, dependency)
	}()
}

// - Aliases

// MaxBytesReader limits the size of a reader
func (r Request) MaxBytesReader(rc io.ReadCloser, n int64) io.ReadCloser {
	return http.MaxBytesReader(r.w, rc, n)
}

// ServeContent replies to the request using the content in the provided ReadSeeker
func (r Request) ServeContent(name string, modtime time.Time, content io.ReadSeeker) {
	r.HandleBody(func(w http.ResponseWriter) {
		http.ServeContent(w, r.Request, name, modtime, content)
	})
}

// ServeFile replies to the request with the contents of the named file or directory
func (r Request) ServeFile(name string) {
	r.HandleBody(func(w http.ResponseWriter) {
		http.ServeFile(w, r.Request, name)
	})
}

// SetCookie adds a Set-Cookie header to the provided ResponseWriter's headers
func (r Request) SetCookie(cookie *http.Cookie) {
	r.Lock()
	defer r.Unlock()
	http.SetCookie(r.w, cookie)
}
