package wok

import (
	"net/http"
)

type LocalApp struct {
	Router
}

type ControllerFunc func(r *http.Request, cr ControllerRequest)

func NewLocalApp(router Router) *LocalApp {
	return &LocalApp{router}
}

func (a *LocalApp) Delegate(prefix string, app App) {
	// TODO
}

func (a *LocalApp) Controller(name string, fn ControllerFunc) {
	// TODO
}

func (a *LocalApp) Run(r *http.Request, controllerRequests []ControllerRequest) {
	// TODO
}
