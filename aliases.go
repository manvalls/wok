package wok

import (
	"github.com/manvalls/wit"
)

// Command applies given commands directly
func Command(commands ...wit.Command) Options {
	return DefaultOptions.Command(commands...)
}

// Run applies the command returned by the provided function
func Run(fn func(r Request) wit.Command) Options {
	return DefaultOptions.Run(fn)
}

// Do does something with the request without returning a delta
func Do(fn func(r ReadOnlyRequest)) Options {
	return DefaultOptions.Do(fn)
}

// Tap does something with the request without returning a delta
func Tap(fn func(r ReadOnlyRequest)) Options {
	return DefaultOptions.Tap(fn)
}

// Handle always applies the command returned by the provided function
func Handle(fn func(r Request) wit.Command) Options {
	return DefaultOptions.Handle(fn)
}

// Sync runs plans sequentially
func Sync() Options {
	return DefaultOptions.Sync()
}

// Excl runs plans exclusively, no other plan is allowed
// to run at the same time
func Excl() Options {
	return DefaultOptions.Excl()
}

// Always runs plans even if it wouldn't be necessary
func Always() Options {
	return DefaultOptions.Always()
}

// With makes the given list of parameters available to the derived plans
func With(params ...string) Options {
	return DefaultOptions.With(params...)
}

// Navigation runs plans on navigation
func Navigation() Options {
	return DefaultOptions.Navigation()
}

// AJAX runs plans on AJAX
func AJAX() Options {
	return DefaultOptions.AJAX()
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

// Socket runs plans on socket request
func Socket() Options {
	return DefaultOptions.Socket()
}

// NoSocket doesn't run plans on socket request
func NoSocket() Options {
	return DefaultOptions.NoSocket()
}

// Call runs this plan when the request matches one of the provided calls
func Call(calls ...string) Options {
	return DefaultOptions.Call(calls...)
}

// NoCall runs this plan when the request doesn't match one of the provided calls
func NoCall(calls ...string) Options {
	return DefaultOptions.NoCall(calls...)
}
