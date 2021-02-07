package wok

import "net/http"

type Handler struct {
	Router
	App
}

type LocalHandler struct {
	Handler
	*LocalRouter
	*LocalApp
}

func NewHandler() LocalHandler {
	router := NewLocalRouter()
	app := NewLocalApp(router)

	return LocalHandler{
		LocalRouter: router,
		LocalApp:    app,
		Handler:     Handler{router, app},
	}
}

func (h Handler) ServeHTTP(http.ResponseWriter, *http.Request) {

}
