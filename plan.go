package wok

import (
	"net/http"

	"github.com/manvalls/wit"
)

type plan struct {
	fn     func(r Request) wit.Action
	action wit.Action
	deps   func(string) wit.Action
	Options
}

// Procedure describes how to handle a certain request
type Procedure struct {
	plans []plan
}

// Procedure returns this procedure itself
func (p Procedure) Procedure() Procedure {
	return p
}

// Plan encapsulates a procedure
type Plan interface {
	Procedure() Procedure
}

// Options holds the list of options selected for a given plan
type Options struct {
	async       bool
	exclusive   bool
	handler     bool
	navigation  bool
	ajax        bool
	params      []string
	methods     map[string]bool
	exclMethods map[string]bool
}

// DefaultOptions are the options which apply to plans by default
var DefaultOptions = Options{
	async:      true,
	navigation: true,
}

// List groups several plans together
func List(plans ...Plan) Plan {
	list := []plan{}

	for _, plan := range plans {
		for _, c := range plan.Procedure().plans {
			list = append(list, c)
		}
	}

	return Procedure{list}
}

// Wrap applies these options to the provided list of plans
func (o Options) Wrap(plans ...Plan) Plan {
	list := []plan{}

	for _, plan := range plans {
		for _, c := range plan.Procedure().plans {
			c.Options = o
			list = append(list, c)
		}
	}

	return Procedure{list}
}

// Action applies given actions directly
func (o Options) Action(actions ...wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				action:  wit.List(actions...),
				Options: o,
			},
		},
	}
}

// Run applies the action returned by the provided function
func (o Options) Run(fn func(r Request) wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				fn:      fn,
				Options: o,
			},
		},
	}
}

// Handle always applies the action returned by the provided function
func (o Options) Handle(fn func(r Request) wit.Action) Procedure {
	o.navigation = true
	o.ajax = true
	o.handler = true
	return o.Run(fn)
}

// Sync runs plans sequentially
func (o Options) Sync() Options {
	o.async = true
	return o
}

// Async runs plans in parallel
func (o Options) Async() Options {
	o.async = true
	return o
}

// Excl runs plans exclusively, no other plan is allowed
// to run at the same time
func (o Options) Excl() Options {
	o.exclusive = true
	return o
}

// Incl runs plans inclusively
func (o Options) Incl() Options {
	o.exclusive = false
	return o
}

// Always runs plans even if it wouldn't be necessary
func (o Options) Always() Options {
	o.handler = true
	return o
}

// WhenNeeded runs plans only when it's needed
func (o Options) WhenNeeded() Options {
	o.handler = false
	return o
}

// With makes the given list of parameters available to the derived plans
func (o Options) With(params ...string) Options {
	newParams := make([]string, len(o.params), len(o.params)+len(params))
	if len(newParams) > 0 {
		copy(newParams, o.params)
	}

	o.params = append(o.params, params...)
	return o
}

// SetParams sets the available parameters to the provided list
func (o Options) SetParams(params ...string) Options {
	o.params = params
	return o
}

// Navigation runs plans on navigation
func (o Options) Navigation() Options {
	o.navigation = true
	return o
}

// NavigationOnly runs plans only on navigation
func (o Options) NavigationOnly() Options {
	o.ajax = false
	return o
}

// AJAX runs plans on AJAX
func (o Options) AJAX() Options {
	o.ajax = true
	return o
}

// AJAXOnly runs plans only on AJAX
func (o Options) AJAXOnly() Options {
	o.navigation = false
	return o
}

func copyMethodMap(methods map[string]bool) map[string]bool {
	m := make(map[string]bool, len(methods))
	for method, value := range methods {
		m[method] = value
	}

	return m
}

func addMethods(methodsMap map[string]bool, methodsList []string) {
	for _, method := range methodsList {
		methodsMap[method] = true
	}
}

func deleteMethods(methodsMap map[string]bool, methodsList []string) {
	if len(methodsMap) == 0 {
		return
	}

	for _, method := range methodsList {
		delete(methodsMap, method)
	}
}

// ResetMethods clears methods blacklist and whitelist
func (o Options) ResetMethods() Options {
	o.methods = nil
	o.exclMethods = nil
	return o
}

// Method runs this plan when the request matches one of the provided methods
func (o Options) Method(methods ...string) Options {
	o.methods = copyMethodMap(o.methods)
	o.exclMethods = copyMethodMap(o.exclMethods)

	addMethods(o.methods, methods)
	deleteMethods(o.exclMethods, methods)
	return o
}

// NoMethod runs this plan when the request doesn't match one of the provided methods
func (o Options) NoMethod(methods ...string) Options {
	o.methods = copyMethodMap(o.methods)
	o.exclMethods = copyMethodMap(o.exclMethods)

	deleteMethods(o.methods, methods)
	addMethods(o.exclMethods, methods)
	return o
}

// Get is an alias for Method("GET", "HEAD")
func (o Options) Get() Options {
	return o.Method(http.MethodGet, http.MethodHead)
}

// NoGet is an alias for NoMethod("GET", "HEAD")
func (o Options) NoGet() Options {
	return o.NoMethod(http.MethodGet, http.MethodHead)
}

// Post is an alias for Method("POST")
func (o Options) Post() Options {
	return o.Method(http.MethodPost)
}

// NoPost is an alias for NoMethod("POST")
func (o Options) NoPost() Options {
	return o.NoMethod(http.MethodPost)
}

// Put is an alias for Method("PUT")
func (o Options) Put() Options {
	return o.Method(http.MethodPut)
}

// NoPut is an alias for NoMethod("PUT")
func (o Options) NoPut() Options {
	return o.NoMethod(http.MethodPut)
}

// Patch is an alias for Method("PATCH")
func (o Options) Patch() Options {
	return o.Method(http.MethodPatch)
}

// NoPatch is an alias for NoMethod("PATCH")
func (o Options) NoPatch() Options {
	return o.NoMethod(http.MethodPatch)
}

// Delete is an alias for Method("DELETE")
func (o Options) Delete() Options {
	return o.Method(http.MethodDelete)
}

// NoDelete is an alias for NoMethod("DELETE")
func (o Options) NoDelete() Options {
	return o.NoMethod(http.MethodDelete)
}

// Deps handles loaded dependencies
func Deps(handler func(string) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				deps: handler,
			},
		},
	}
}

// Nil represents an effectless plan
var Nil Plan = Procedure{}

// IsNil checks whether the given plan is empty or not
func IsNil(plan Plan) bool {
	return len(plan.Procedure().plans) == 0
}
