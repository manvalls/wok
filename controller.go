package wok

import "github.com/manvalls/wit"

type controller struct {
	fn        func(r Request) wit.Delta
	delta     wit.Delta
	async     bool
	exclusive bool
	handler   bool
	params    []string
}

// Controller describes how to handle a certain request
type Controller struct {
	controllers []controller
}

// List groups several controllers together
func List(controllers ...Controller) Controller {
	list := []controller{}

	for _, controller := range controllers {
		for _, c := range controller.controllers {
			list = append(list, c)
		}
	}

	return Controller{list}
}

// Delta applies the given delta directly
func Delta(deltas ...wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				delta: wit.List(deltas...),
				async: true,
			},
		},
	}
}

// Async handles the given request in parallel with other async controllers
func Async(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:    fn,
				async: true,
			},
		},
	}
}

// Sync handles the given request sequentially
func Sync(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn: fn,
			},
		},
	}
}

// Excl handles the given request exclusively, no other controller is allowed
// to run at the same time
func Excl(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:        fn,
				exclusive: true,
			},
		},
	}
}

// AsyncHandler is always run at the current step no matter what the previous state was,
// in parallel
func AsyncHandler(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
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
func SyncHandler(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:      fn,
				handler: true,
			},
		},
	}
}

// ExclHandler is always run at the current step no matter what the previous state was,
// exclusively
func ExclHandler(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
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

// With makes the given list of parameters available to the derived controllers
func With(params ...string) ParamsWrapper {
	return ParamsWrapper{params}
}

// Async handles the given request in parallel with other async controllers
func (wp ParamsWrapper) Async(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:     fn,
				async:  true,
				params: wp.params,
			},
		},
	}
}

// Sync handles the given request sequentially
func (wp ParamsWrapper) Sync(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:     fn,
				params: wp.params,
			},
		},
	}
}

// Excl handles the given request exclusively, no other controller is allowed
// to run at the same time
func (wp ParamsWrapper) Excl(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
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
func (wp ParamsWrapper) AsyncHandler(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
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
func (wp ParamsWrapper) SyncHandler(fn func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				fn:      fn,
				handler: true,
				params:  wp.params,
			},
		},
	}
}

// DeltaHandler applies the given delta directly no matter what the previous state was
func DeltaHandler(deltas ...wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				delta:   wit.List(deltas...),
				async:   true,
				handler: true,
			},
		},
	}
}

// Nil represents an effectless controller
var Nil = Controller{}

// IsNil checks whether the given controller is empty or not
func IsNil(controller Controller) bool {
	return len(controller.controllers) == 0
}
