package wok

import (
	"context"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/manvalls/way"
	"github.com/manvalls/wit"
)

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

// Request holds a list of useful objects together
type Request struct {
	*StatusCodeGetterSetter
	http.ResponseWriter
	*http.Request
	Deduper
	Runner
	*Scope
	way.Router
	RequestHeader  http.Header
	ResponseHeader http.Header
}

// NewRunner builds a new runner linked to this request
func (r Request) NewRunner(handler func(r Request), header string, params Params, route ...uint) {
	r.Runner.NewRunner(func(runner Runner) {
		r.Runner = runner
		handler(r)
	}, header, params, route...)
}

// NewDeduper builds a new deduper linked to this request
func (r Request) NewDeduper(header string) Request {
	r.Deduper = r.Scope.NewDeduper(header)
	return r
}

// Run executes the given function, if needed
func (r Request) Run(f func(ctx context.Context) wit.Delta) {
	r.Runner.Run(f)
}

// Next returns a new request for the next step
func (r Request) Next(handler func(r Request)) {
	r.Runner.Next(func(runner Runner) {
		r.Runner = runner
		handler(r)
	})
}

// URLRedirect issues an HTTP redirection
func (r Request) URLRedirect(statusCode int, params way.Params, route ...uint) wit.Delta {
	redirURL, err := r.GetURL(params, route...)
	if err != nil {
		return wit.Error(err)
	}

	r.ResponseHeader.Set("Location", redirURL)
	r.SetStatusCode(statusCode)
	return wit.End
}

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

// Handler implements an HTTP handler which provides wok requests
type Handler struct {
	Handler       func(r Request)
	RunnerHeader  string
	DeduperHeader string
	way.Router
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, route, err := h.GetRoute(r.URL)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	scope := NewScope(r)

	runnerHeader := h.RunnerHeader
	if runnerHeader == "" {
		runnerHeader = "X-Wok-Runner"
	}

	deduperHeader := h.DeduperHeader
	if deduperHeader == "" {
		deduperHeader = "X-Wok-Deduper"
	}

	sc := &StatusCodeGetterSetter{}
	err = scope.Write(w, scope.NewRunner(func(runner Runner) {
		h.Handler(Request{
			sc,
			w,
			r,
			scope.NewDeduper(deduperHeader),
			runner,
			scope,
			h.Router,
			r.Header,
			w.Header(),
		})
	}, runnerHeader, params, route...), sc)

	if err != nil {
		w.WriteHeader(500)
		fmt.Fprintln(w, err)
		return
	}
}
