package wok

import (
	"net/http"
	"net/url"

	"github.com/manvalls/wit"
)

// Params holds the list of parameters for a given URL
type Params = url.Values

// Router computes the list of controllers that should run for a given URL or route
type Router interface {
	ResolveURL(r *http.Request, url *url.URL) (result RouteResult)
	ResolveRoute(r *http.Request, route string, params Params) (resolvedURL string)
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

	Persistent    bool // Things that stay in the page even after navigating away
	Lazy          bool // When true, don't run on the initial page load, run them from the client instead
	Socket        bool // If true, use a real-time websocket
	HasValidation bool // If true, dry-run when filling out the form
	Cache         bool // If true, cache the result for subsequent requests
}

// ControllerRequest represents a request to run a controller
type ControllerRequest interface {
	Controller() string
	DryRun() bool
	Socket() bool
	Params() Params

	Redirect(url string, status int) ControllerRequest
	ExternalRedirect(url string, status int) ControllerRequest
	SetStatus(status int) ControllerRequest

	AddHeader(key string, values ...string) ControllerRequest
	SetHeader(key string, values ...string) ControllerRequest

	ReloadOn(events ...string) ControllerRequest
	AbortOn(events ...string) ControllerRequest

	SendDelta(delta wit.Delta) ControllerRequest

	WaitUntil(flags ...string) ControllerRequest
	WaitUntilNot(flags ...string) ControllerRequest

	Trigger(events ...string) ControllerRequest
	Set(flags ...string) ControllerRequest
	Unset(flags ...string) ControllerRequest

	Cleanup() Cleanup

	Close()
}

type Cleanup interface {
	SendDelta(delta wit.Delta) Cleanup

	WaitUntil(flags ...string) Cleanup
	WaitUntilNot(flags ...string) Cleanup

	Trigger(events ...string) Cleanup
	Set(flags ...string) Cleanup
	Unset(flags ...string) Cleanup
}

// App holds and runs the list of controllers that conform an application
type App interface {
	Run(r *http.Request, controllerRequest ControllerRequest)
}
