package wok

import (
	"net/http"
	"strings"
)

type LocalApp struct {
	Router
	controllers map[string]ControllerFunc
}

type ControllerFunc func(r *http.Request, cr ControllerRequest)

func NewLocalApp(router Router) *LocalApp {
	return &LocalApp{
		router,
		map[string]ControllerFunc{},
	}
}

func (a *LocalApp) Controller(name string, fn ControllerFunc) {
	a.controllers[name] = fn
}

func (a *LocalApp) Run(r *http.Request, controllerRequest ControllerRequest) {
	if fn, ok := a.controllers[controllerRequest.Controller()]; ok {
		fn(r, controllerRequest)
		return
	}

	parts := strings.Split(controllerRequest.Controller(), ".")

	for len(parts) > 0 {
		parts = parts[0 : len(parts)-1]
		prefix := strings.Join(parts, ".")

		if fn, ok := a.controllers[prefix]; ok {
			fn(r, controllerRequest)
			return
		}
	}
}
