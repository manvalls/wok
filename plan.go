package wok

import (
	"net/http"

	"github.com/manvalls/wit"
)

type linkedPlan struct {
	plan   Plan
	parent *linkedPlan
}

type plan struct {
	fn      func(r Request) wit.Command
	doFn    func(r ReadOnlyRequest)
	command wit.Command
	deps    func(string) wit.Command
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

const (
	unsetField = iota
	trueField
	falseField
)

// Options holds the list of options selected for a given plan
type Options struct {
	sync        bool
	exclusive   bool
	handler     bool
	navigation  bool
	ajax        bool
	socket      int
	params      []string
	methods     map[string]bool
	exclMethods map[string]bool
	calls       map[string]bool
	exclCalls   map[string]bool
	*linkedPlan
}

// DefaultOptions are the options which apply to plans by default
var DefaultOptions = Options{}

// List groups several plans together
func List(plans ...Plan) Plan {
	list := []plan{}

	for _, plan := range plans {
		if plan != nil {
			for _, c := range plan.Procedure().plans {
				list = append(list, c)
			}
		}
	}

	return Procedure{list}
}

// Procedure returns the computed procedure
func (o Options) Procedure() Procedure {
	reverseOrderPlans := []Plan{}

	linkedPlan := o.linkedPlan
	for linkedPlan != nil {
		reverseOrderPlans = append(reverseOrderPlans, linkedPlan.plan)
		linkedPlan = linkedPlan.parent
	}

	plans := make([]Plan, len(reverseOrderPlans))
	for i, plan := range reverseOrderPlans {
		plans[len(plans)-1-i] = plan
	}

	return List(plans...).Procedure()
}

// Command applies given commands directly
func (o Options) Command(commands ...wit.Command) Options {
	r := o

	if o.navigation == false && o.ajax == false && o.socket != trueField {
		o.navigation = true
	}

	r.linkedPlan = &linkedPlan{
		parent: r.linkedPlan,
		plan: Procedure{
			plans: []plan{
				{
					command: wit.List(commands...),
					Options: o,
				},
			},
		},
	}

	return r
}

// Run applies the command returned by the provided function
func (o Options) Run(fn func(r Request) wit.Command) Options {
	r := o

	if o.navigation == false && o.ajax == false && o.socket != trueField {
		o.navigation = true
	}

	r.linkedPlan = &linkedPlan{
		parent: r.linkedPlan,
		plan: Procedure{
			plans: []plan{
				{
					fn:      fn,
					Options: o,
				},
			},
		},
	}

	return r
}

// Handle always applies the command returned by the provided function
func (o Options) Handle(fn func(r Request) wit.Command) Options {
	if o.navigation == false && o.ajax == false && o.socket != trueField {
		o.navigation = true
		o.ajax = true
	}

	o.handler = true
	return o.Run(fn)
}

// Do always does something with the request without returning a delta
func (o Options) Do(fn func(r ReadOnlyRequest)) Options {
	if o.navigation == false && o.ajax == false && o.socket != trueField {
		o.navigation = true
		o.ajax = true
	}

	o.handler = true
	return o.Tap(fn)
}

// Tap does something with the request without returning a delta
func (o Options) Tap(fn func(r ReadOnlyRequest)) Options {
	r := o

	if o.navigation == false && o.ajax == false && o.socket != trueField {
		o.navigation = true
	}

	r.linkedPlan = &linkedPlan{
		parent: r.linkedPlan,
		plan: Procedure{
			plans: []plan{
				{
					doFn:    fn,
					Options: o,
				},
			},
		},
	}

	return r
}

// Sync runs plans sequentially
func (o Options) Sync() Options {
	o.sync = true
	return o
}

// Excl runs plans exclusively, no other plan is allowed
// to run at the same time
func (o Options) Excl() Options {
	o.exclusive = true
	return o
}

// Always runs plans even if it wouldn't be necessary
func (o Options) Always() Options {
	o.handler = true
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

// Navigation runs plans on navigation
func (o Options) Navigation() Options {
	o.navigation = true
	return o
}

// AJAX runs plans on AJAX
func (o Options) AJAX() Options {
	o.ajax = true
	return o
}

// Socket runs plans on socket request
func (o Options) Socket() Options {
	o.socket = trueField
	return o
}

// NoSocket doesn't run plans on socket request
func (o Options) NoSocket() Options {
	o.socket = falseField
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

// Call runs this plan when the request matches one of the provided calls
func (o Options) Call(calls ...string) Options {
	o.calls = copyMethodMap(o.calls)
	o.exclCalls = copyMethodMap(o.exclCalls)

	addMethods(o.calls, calls)
	deleteMethods(o.exclCalls, calls)
	return o
}

// NoCall runs this plan when the request doesn't match one of the provided calls
func (o Options) NoCall(calls ...string) Options {
	o.calls = copyMethodMap(o.calls)
	o.exclCalls = copyMethodMap(o.exclCalls)

	deleteMethods(o.calls, calls)
	addMethods(o.exclCalls, calls)
	return o
}

// Deps handles loaded dependencies
func Deps(handler func(string) wit.Command) Plan {
	return Procedure{
		plans: []plan{
			{
				deps: handler,
				Options: Options{
					navigation: true,
					ajax:       true,
				},
			},
		},
	}
}

// Nil represents an effectless plan
var Nil Plan = Procedure{}

// IsNil checks whether the given plan is empty or not
func IsNil(plan Plan) bool {
	return plan == nil || len(plan.Procedure().plans) == 0
}
