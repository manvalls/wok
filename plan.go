package wok

import "github.com/manvalls/wit"

type plan struct {
	fn        func(r Request) wit.Action
	action    wit.Action
	async     bool
	exclusive bool
	handler   bool
	params    []string
	deps      func(string) wit.Action
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

// List groups several plans together
func List(plans ...Plan) Procedure {
	list := []plan{}

	for _, plan := range plans {
		for _, c := range plan.Procedure().plans {
			list = append(list, c)
		}
	}

	return Procedure{list}
}

// Action applies given actions directly
func Action(actions ...wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				action: wit.List(actions...),
				async:  true,
			},
		},
	}
}

// Run applies the action returned by the provided function
func Run(fn func(r Request) wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				fn:    fn,
				async: true,
			},
		},
	}
}

// Handle always applies the action returned by the provided function
func Handle(fn func(r Request) wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				fn:      fn,
				async:   true,
				handler: true,
			},
		},
	}
}

// Sync marks the current plan as sequential
func (p Procedure) Sync() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.async = false
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// Async marks the current plan as asynchronous
func (p Procedure) Async() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.async = true
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// Excl marks the current plan as exclusive, no other plan is allowed
// to run at the same time
func (p Procedure) Excl() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.exclusive = true
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// Incl marks the current plan as inclusive
func (p Procedure) Incl() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.exclusive = false
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// Always runs the current plan even if it wouldn't be necessary
func (p Procedure) Always() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.handler = true
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// WhenNeeded runs the current plan only when it's needed
func (p Procedure) WhenNeeded() Procedure {
	plans := make([]plan, len(p.plans))

	for i, plan := range p.plans {
		plan.handler = false
		plans[i] = plan
	}

	return Procedure{plans: plans}
}

// ParamsWrapper holds a list of request parameters
type ParamsWrapper struct {
	params []string
}

// With makes the given list of parameters available to the derived plans
func With(params ...string) ParamsWrapper {
	return ParamsWrapper{params}
}

// Run applies the action returned by the provided function
func (wp ParamsWrapper) Run(fn func(r Request) wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				fn:     fn,
				async:  true,
				params: wp.params,
			},
		},
	}
}

// Handle always applies the action returned by the provided function
func (wp ParamsWrapper) Handle(fn func(r Request) wit.Action) Procedure {
	return Procedure{
		plans: []plan{
			{
				fn:      fn,
				async:   true,
				handler: true,
				params:  wp.params,
			},
		},
	}
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
var Nil = Procedure{}

// IsNil checks whether the given plan is empty or not
func IsNil(plan Plan) bool {
	return len(plan.Procedure().plans) == 0
}
