package wok

import "github.com/manvalls/wit"

type plan struct {
	fn        func(r Request) wit.Action
	action    wit.Action
	async     bool
	exclusive bool
	handler   bool
	params    []string
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
func List(plans ...Plan) Plan {
	list := []plan{}

	for _, plan := range plans {
		for _, c := range plan.Procedure().plans {
			list = append(list, c)
		}
	}

	return Procedure{list}
}

// Action applies given actions directly
func Action(actions ...wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				action: wit.List(actions...),
				async:  true,
			},
		},
	}
}

// Async handles the given request in parallel with other async plans
func Async(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:    fn,
				async: true,
			},
		},
	}
}

// Sync handles the given request sequentially
func Sync(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn: fn,
			},
		},
	}
}

// Excl handles the given request exclusively, no other plan is allowed
// to run at the same time
func Excl(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:        fn,
				exclusive: true,
			},
		},
	}
}

// AsyncHandler is always run at the current step no matter what the previous state was,
// in parallel
func AsyncHandler(fn func(r Request) wit.Action) Plan {
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

// SyncHandler is always run at the current step no matter what the previous state was,
// sequentially
func SyncHandler(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:      fn,
				handler: true,
			},
		},
	}
}

// ExclHandler is always run at the current step no matter what the previous state was,
// exclusively
func ExclHandler(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:        fn,
				handler:   true,
				exclusive: true,
			},
		},
	}
}

// ParamsWrapper holds a list of request parameters
type ParamsWrapper struct {
	params []string
}

// With makes the given list of parameters available to the derived plans
func With(params ...string) ParamsWrapper {
	return ParamsWrapper{params}
}

// Async handles the given request in parallel with other async plans
func (wp ParamsWrapper) Async(fn func(r Request) wit.Action) Plan {
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

// Sync handles the given request sequentially
func (wp ParamsWrapper) Sync(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:     fn,
				params: wp.params,
			},
		},
	}
}

// Excl handles the given request exclusively, no other plan is allowed
// to run at the same time
func (wp ParamsWrapper) Excl(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:        fn,
				params:    wp.params,
				exclusive: true,
			},
		},
	}
}

// AsyncHandler is always run at the current step no matter what the previous state was,
// in parallel
func (wp ParamsWrapper) AsyncHandler(fn func(r Request) wit.Action) Plan {
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

// SyncHandler is always run at the current step no matter what the previous state was,
// sequentially
func (wp ParamsWrapper) SyncHandler(fn func(r Request) wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				fn:      fn,
				handler: true,
				params:  wp.params,
			},
		},
	}
}

// ActionHandler applies given actions directly no matter what the previous state was
func ActionHandler(actions ...wit.Action) Plan {
	return Procedure{
		plans: []plan{
			{
				action:  wit.List(actions...),
				async:   true,
				handler: true,
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
