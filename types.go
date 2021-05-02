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
	Persistent    bool     // Things that stay in the page even after navigating away - only re-applied when DependsOn are invalidated
	Lazy          bool     // When true, don't run on the initial page load, run them from the client instead
	Socket        bool     // If true, use a real-time websocket
	NeedsCleanup  bool     // If true, run the cleanup function when navigating away
	HasValidation bool     // If true, dry-run when filling out the form
	Cache         bool     // If true, cache the result for subsequent requests
}

// ControllerRequest represents a request to run a controller
type ControllerRequest interface {
	Controller() string
	Cleanup() bool
	DryRun() bool
	Params() Params

	SendDelta(delta wit.Delta)
	Redirect(url string, status int)
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
	Run(r *http.Request, controllerRequest ControllerRequest)
}
