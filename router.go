package wok

import (
	"net/http"
)

type LocalRouter struct{}

func NewLocalRouter() *LocalRouter {
	return &LocalRouter{}
}

func (r *LocalRouter) ResolveURL(req *http.Request, url string) RouteResult {
	return RouteResult{}
}

func (r *LocalRouter) ResolveRoute(req *http.Request, route string, params map[string]string) (resolvedURL string, result RouteResult) {
	return "", RouteResult{}
}
