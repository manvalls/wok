package wok

import "github.com/manvalls/wit"

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
	async     bool
	exclusive bool
	handler   bool
	params    []string
}

// DefaultOptions are the options which apply to plans by default
var DefaultOptions = Options{
	async: true,
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
	return o.Always().Run(fn)
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
