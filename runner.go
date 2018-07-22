package wok

import (
	"context"
	"net/http"

	"github.com/manvalls/wit"
)

// Runner runs steps according to previous and current route
type Runner struct {
	index     int
	start     int
	route     []uint
	prevRoute []uint
	scope     *Scope
}

// Runner builds a new runner linked to this scope
func (s *Scope) Runner(header string, route ...uint) Runner {
	header = http.CanonicalHeaderKey(header)
	prevRoute := s.FromHeader(header)

	startIndex := len(prevRoute)
	for i, j := range prevRoute {
		if i >= len(route) || route[i] != j {
			startIndex = i
			break
		}
	}

	s.mutex.Lock()
	s.routes[header] = ToHeader(route...)
	s.mutex.Unlock()

	return Runner{0, startIndex, route, prevRoute, s}
}

// Run executes the given function, if needed
func (r Runner) Run(f func(context.Context) wit.Delta) wit.Delta {
	if r.index >= r.start {
		return wit.Run(r.scope.req.Context(), f)
	}

	return wit.Nil
}

// Route returns the current route step
func (r Runner) Route() uint {
	if r.index >= len(r.route) {
		return 0
	}

	return r.route[r.index]
}

// OldRoute returns the old route for the current step
func (r Runner) OldRoute() uint {
	if r.index >= len(r.prevRoute) {
		return 0
	}

	return r.prevRoute[r.index]
}

// OffLimits returns whether or not we're past the matched path
func (r Runner) OffLimits() bool {
	return r.index >= len(r.route)
}

// OffOldLimits returns whether or not we're past the old path
func (r Runner) OffOldLimits() bool {
	return r.index >= len(r.prevRoute)
}

// NeedsCleanup returns whether or not the previous route needs cleaning up
func (r Runner) NeedsCleanup() bool {
	return !r.OffOldLimits() && (r.OffLimits() || r.Route() != r.OldRoute())
}

// Next returns a new runner for the next step
func (r Runner) Next() Runner {
	r.index++
	return r
}
