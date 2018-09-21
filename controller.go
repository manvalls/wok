package wok

import "github.com/manvalls/wit"

type controller struct {
	handler func(r Request) wit.Delta
	delta   wit.Delta
	async   bool
	setup   bool
	params  []string
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
func Async(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				async:   true,
			},
		},
	}
}

// Sync handles the given request sequentially, no other controller is allowed
// to run at the same time
func Sync(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
			},
		},
	}
}

// AsyncSetup is always run at the current step no matter what the previous state was,
// in parallel
func AsyncSetup(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				async:   true,
				setup:   true,
			},
		},
	}
}

// SyncSetup is always run at the current step no matter what the previous state was,
// sequentially
func SyncSetup(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				setup:   true,
			},
		},
	}
}

// WithParams holds a list of request parameters
type WithParams struct {
	params []string
}

// With makes the given list of parameters available to the derived controllers
func With(params ...string) WithParams {
	return WithParams{params}
}

// Async handles the given request in parallel with other async controllers
func (wp WithParams) Async(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				async:   true,
				params:  wp.params,
			},
		},
	}
}

// Sync handles the given request sequentially, no other controller is allowed
// to run at the same time
func (wp WithParams) Sync(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				params:  wp.params,
			},
		},
	}
}

// AsyncSetup is always run at the current step no matter what the previous state was,
// in parallel
func (wp WithParams) AsyncSetup(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				async:   true,
				setup:   true,
				params:  wp.params,
			},
		},
	}
}

// SyncSetup is always run at the current step no matter what the previous state was,
// sequentially
func (wp WithParams) SyncSetup(handler func(r Request) wit.Delta) Controller {
	return Controller{
		controllers: []controller{
			{
				handler: handler,
				setup:   true,
				params:  wp.params,
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
