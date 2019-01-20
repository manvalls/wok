package wok

import (
	"github.com/manvalls/wit"
)

// Wrap applies default options to the provided list of plans
func Wrap(plans ...Plan) Plan {
	return DefaultOptions.Wrap(plans...)
}

// Action applies given actions directly
func Action(actions ...wit.Action) Plan {
	return DefaultOptions.Action(actions...)
}

// Run applies the action returned by the provided function
func Run(fn func(r Request) wit.Action) Plan {
	return DefaultOptions.Run(fn)
}

// Handle always applies the action returned by the provided function
func Handle(fn func(r Request) wit.Action) Plan {
	return DefaultOptions.Handle(fn)
}

// Sync runs plans sequentially
func Sync() Options {
	return DefaultOptions.Sync()
}

// Async runs plans in parallel
func Async() Options {
	return DefaultOptions.Async()
}

// Excl runs plans exclusively, no other plan is allowed
// to run at the same time
func Excl() Options {
	return DefaultOptions.Excl()
}

// Incl runs plans inclusively
func Incl() Options {
	return DefaultOptions.Incl()
}

// Always runs plans even if it wouldn't be necessary
func Always() Options {
	return DefaultOptions.Always()
}

// WhenNeeded runs plans only when it's needed
func WhenNeeded() Options {
	return DefaultOptions.WhenNeeded()
}

// With makes the given list of parameters available to the derived plans
func With(params ...string) Options {
	return DefaultOptions.With(params...)
}

// SetParams sets the available parameters to the provided list
func SetParams(params ...string) Options {
	return DefaultOptions.SetParams(params...)
}

// Navigation runs plans on navigation
func Navigation() Options {
	return DefaultOptions.Navigation()
}

// NavigationOnly runs plans only on navigation
func NavigationOnly() Options {
	return DefaultOptions.NavigationOnly()
}

// AJAX runs plans on AJAX
func AJAX() Options {
	return DefaultOptions.AJAX()
}

// AJAXOnly runs plans only on AJAX
func AJAXOnly() Options {
	return DefaultOptions.AJAXOnly()
}

// ResetMethods clears methods blacklist and whitelist
func ResetMethods() Options {
	return DefaultOptions.ResetMethods()
}

// Method runs this plan when the request matches one of the provided methods
func Method(methods ...string) Options {
	return DefaultOptions.Method(methods...)
}

// NoMethod runs this plan when the request doesn't match one of the provided methods
func NoMethod(methods ...string) Options {
	return DefaultOptions.NoMethod(methods...)
}

// Get is an alias for Method("GET", "HEAD")
func Get() Options {
	return DefaultOptions.Get()
}

// NoGet is an alias for NoMethod("GET", "HEAD")
func NoGet() Options {
	return DefaultOptions.NoGet()
}

// Post is an alias for Method("POST")
func Post() Options {
	return DefaultOptions.Post()
}

// NoPost is an alias for NoMethod("POST")
func NoPost() Options {
	return DefaultOptions.NoPost()
}

// Put is an alias for Method("PUT")
func Put() Options {
	return DefaultOptions.Put()
}

// NoPut is an alias for NoMethod("PUT")
func NoPut() Options {
	return DefaultOptions.NoPut()
}

// Patch is an alias for Method("PATCH")
func Patch() Options {
	return DefaultOptions.Patch()
}

// NoPatch is an alias for NoMethod("PATCH")
func NoPatch() Options {
	return DefaultOptions.NoPatch()
}

// Delete is an alias for Method("DELETE")
func Delete() Options {
	return DefaultOptions.Delete()
}

// NoDelete is an alias for NoMethod("DELETE")
func NoDelete() Options {
	return DefaultOptions.NoDelete()
}
