package wok

import (
	"context"
	"net/http"

	"github.com/manvalls/way"
	"github.com/manvalls/wit"
)

// Request holds a list of useful objects together
type Request struct {
	http.ResponseWriter
	*http.Request
	Deduper
	Runner
	*Scope
	way.Router
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

// Handler implements an HTTP handler which provides wok requests
type Handler struct {
	Handler       func(r Request)
	RunnerHeader  string
	DeduperHeader string
	way.Router
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, route, _ := h.GetRoute(r.URL)
	scope := NewScope(r)

	runnerHeader := h.RunnerHeader
	if runnerHeader == "" {
		runnerHeader = "X-Wok-Runner"
	}

	deduperHeader := h.DeduperHeader
	if deduperHeader == "" {
		deduperHeader = "X-Wok-Deduper"
	}

	scope.Write(w, scope.NewRunner(func(runner Runner) {
		h.Handler(Request{
			w,
			r,
			scope.NewDeduper(deduperHeader),
			runner,
			scope,
			h.Router,
		})
	}, runnerHeader, params, route...))
}
