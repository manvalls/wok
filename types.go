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
type ControllerRequest struct {
	Controller string
	DryRun     bool
	Socket     bool
	Params     Params

	Redirect         func(url string, status int) ControllerRequest
	ExternalRedirect func(url string, status int) ControllerRequest
	SetStatus        func(status int) ControllerRequest

	AddHeader func(key string, values ...string) ControllerRequest
	SetHeader func(key string, values ...string) ControllerRequest

	ReloadOn func(events ...string) ControllerRequest
	AbortOn  func(events ...string) ControllerRequest

	SendDelta func(delta wit.Delta) ControllerRequest

	WaitUntil    func(flags ...string) ControllerRequest
	WaitUntilNot func(flags ...string) ControllerRequest

	Trigger func(events ...string) ControllerRequest
	Set     func(flags ...string) ControllerRequest
	Unset   func(flags ...string) ControllerRequest

	Cleanup func() Cleanup

	Close func()
}

type Cleanup struct {
	SendDelta func(delta wit.Delta) Cleanup

	WaitUntil    func(flags ...string) Cleanup
	WaitUntilNot func(flags ...string) Cleanup

	Trigger func(events ...string) Cleanup
	Set     func(flags ...string) Cleanup
	Unset   func(flags ...string) Cleanup
}

// App holds and runs the list of controllers that conform an application
type App interface {
	Run(r *http.Request, controllerRequest ControllerRequest)
}
