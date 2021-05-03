package wok

import (
	"net/http"
	"strings"
)

type Handler struct {
	Router
	App
	BasePath string
}

type LocalHandler struct {
	Handler
	*LocalRouter
	*LocalApp
}

func NewHandler(basePath string) LocalHandler {
	router := NewLocalRouter()
	app := NewLocalApp(router, basePath)

	return LocalHandler{
		LocalRouter: router,
		LocalApp:    app,
		Handler: Handler{
			Router: router,
			App:    app,
		},
	}
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.BasePath) {
		return
	}

	url := r.URL
	url.Path = url.Path[len(h.BasePath):]

	result := h.ResolveURL(r, url)

}
