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

	DependsOn     []string // E.g, "main", "header" and so on - when these are re-rendered, current controller needs to be re-rendered as well, and when cleaned up, the same
	RunAfter      []string // If these controllers are to be ran, call the current one after getting the response from them
	Persistent    bool     // Things that stay in the page even after navigating away - only re-applied when DependsOn are invalidated
	Batch         bool     // All controllers with Batch: true are grouped together in a single POST request
	Lazy          bool     // When true, don't run on the initial page load, run them from the client instead
	Socket        bool     // If true, use a real-time websocket
	NeedsCleanup  bool     // If true, run the cleanup function when navigating away
	HasValidation bool     // If true, run the validation function before calling the controller
	Cache         bool     // If true, cache the result for subsequent requests
	Prefetch      bool     // If true, include along with the parent request when exposed
}

// ControllerRequest represents a request to run a controller
type ControllerRequest interface {
	Controller() string
	RunAfter() []string
	Cleanup() bool
	DryRun() bool
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
}

// App holds and runs the list of controllers that conform an application
type App interface {
	Run(r *http.Request, controllerRequests []ControllerRequest)
}
