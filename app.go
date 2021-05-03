package wok

import (
	"net/http"
	"strings"

	"github.com/manvalls/wit"
	"github.com/manvalls/wq"
)

type LocalApp struct {
	Router

	basePath    string
	controllers map[string]ControllerFunc
}

type Request struct {
	*http.Request
	ControllerRequest
	wq.Node

	basePath string
	router   Router
}

func (r Request) Route(route string, params Params) (resolvedURL string) {
	return r.basePath + r.router.ResolveRoute(r.Request, route, params)
}

type ControllerFunc func(r Request)

func NewLocalApp(router Router, basePath string) *LocalApp {
	return &LocalApp{
		router,
		basePath,
		map[string]ControllerFunc{},
	}
}

func (a *LocalApp) Controller(name string, fn ControllerFunc) {
	a.controllers[name] = fn
}

func (a *LocalApp) Run(r *http.Request, controllerRequest ControllerRequest) {
	req := Request{
		r,
		controllerRequest,
		wq.Node{
			Send: func(d wit.Delta) {
				controllerRequest.SendDelta(d)
			},
		},
		a.basePath,
		a.Router,
	}

	if fn, ok := a.controllers[controllerRequest.Controller()]; ok {
		fn(req)
		return
	}

	parts := strings.Split(controllerRequest.Controller(), ".")

	for len(parts) > 0 {
		parts = parts[0 : len(parts)-1]
		prefix := strings.Join(parts, ".")

		if fn, ok := a.controllers[prefix]; ok {
			fn(req)
			return
		}
	}
}
