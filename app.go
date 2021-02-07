package wok

import (
	"net/http"
)

type LocalApp struct {
	Router
}

func NewLocalApp(router Router) *LocalApp {
	return &LocalApp{router}
}

func (a *LocalApp) Run(r *http.Request, controllerRequests []ControllerRequest) {

}
