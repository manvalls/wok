package wok

import (
	"context"
	"net/http"
	"net/url"

	"github.com/manvalls/wit"
)

// Params holds the list of parameters for a given URL
type Params = url.Values

// Router computes the list of controllers that should run for a given URL or route
type Router interface {
	ResolveURL(r *http.Request, url *url.URL) (result RouteResult)
	ResolveRoute(r *http.Request, route string, params Params) (resolvedURL string, result RouteResult)
}

// RouteResult contains the resolved plan for a route
type RouteResult struct {
	Controllers []ControllerPlan
	ReloadOn    []string
}

// ControllerPlan represents a controller to be ran
type ControllerPlan struct {
	Controller string
	Method     string
	Params

	DependsOn       []string
	RunAfter        []string
	Batch           bool
	Persistent      bool
	Lazy            bool
	Socket          bool
	NeedsCleanup    bool
	NeedsValidation bool
}

// ControllerRequest represents a request to run a controller
type ControllerRequest interface {
	Controller() string
	RunAfter() []string
	Cleanup() bool
	Validate() bool
	Params() Params

	SendDelta(delta wit.Delta)
	ExposeURL(url string, routeResult RouteResult)
	Redirect(url string, status int, routeResult RouteResult)
	ExternalRedirect(url string, status int)
	SetStatus(status int)
	AddHeader(key string, values []string)
	SetHeader(key string, values []string)
	ReloadOn(events []string)
	AbortOn(events []string)
	Trigger(events []string)
	Close()

	context.Context
}

// App holds and runs the list of controllers that conform an application
type App interface {
	Run(r *http.Request, controllerRequests []ControllerRequest)
}
