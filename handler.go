package wok

import "net/http"

type Handler struct {
	Router
	App
	BaseURL string
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
		Handler: Handler{
			Router: router,
			App:    app,
		},
	}
}

func (h Handler) ServeHTTP(http.ResponseWriter, *http.Request) {

}
