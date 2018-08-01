package wok

import (
	"context"
	"net/http"
	"net/url"
	"sync"

	"github.com/manvalls/wit"
)

type deltaAggregator struct {
	sync.Mutex
	deltas []wit.Delta
}

// Runner runs steps according to previous and current route
type Runner struct {
	index      int
	start      int
	params     Params
	route      []uint
	prevParams Params
	prevRoute  []uint
	scope      *Scope
	aggregator *deltaAggregator
}

// Runner builds a new runner linked to this scope
func (s *Scope) Runner(header string, params Params, route ...uint) *Runner {
	header = http.CanonicalHeaderKey(header)
	prevParams, prevRoute := s.FromHeader(header)

	startIndex := len(prevRoute)
	for i, j := range prevRoute {
		if i >= len(route) || route[i] != j {
			startIndex = i
			break
		}
	}

	s.mutex.Lock()
	s.routes[header] = ToHeader(params, route...)
	s.mutex.Unlock()

	return &Runner{0, startIndex, params, route, prevParams, prevRoute, s, &deltaAggregator{
		sync.Mutex{},
		[]wit.Delta{},
	}}
}

// Run executes the given function, if needed
func (r Runner) Run(f func(context.Context) wit.Delta) {
	if r.index >= r.start {
		delta := wit.Run(r.scope.req.Context(), f)

		r.aggregator.Lock()
		r.aggregator.deltas = append(r.aggregator.deltas, delta)
		r.aggregator.Unlock()
	}
}

// RunParams executes the given function with the given params, if needed
func (r Runner) RunParams(f func(context.Context, url.Values) wit.Delta, params ...string) {
	equal := r.index < r.start
	filteredParams := Params{}

	for _, param := range params {
		newParam := r.params[param]
		oldParam := r.prevParams[param]

		filteredParams[param] = newParam

		if equal {
			if len(newParam) != len(oldParam) {
				equal = false
			} else {
				for i := range newParam {
					if newParam[i] != oldParam[i] {
						equal = false
						break
					}
				}
			}
		}
	}

	if !equal {
		delta := wit.Run(r.scope.req.Context(), func(ctx context.Context) wit.Delta {
			return f(ctx, filteredParams)
		})

		r.aggregator.Lock()
		r.aggregator.deltas = append(r.aggregator.deltas, delta)
		r.aggregator.Unlock()
	}
}

// Append appends deltas to the internal buffer
func (r Runner) Append(deltas ...wit.Delta) {
	r.aggregator.Lock()
	r.aggregator.deltas = append(r.aggregator.deltas, deltas...)
	r.aggregator.Unlock()
}

// Delta flushes the internal buffer to the returned delta
func (r Runner) Delta() wit.Delta {
	r.aggregator.Lock()
	defer r.aggregator.Unlock()

	delta := wit.List(r.aggregator.deltas...)
	r.aggregator.deltas = []wit.Delta{}
	return delta
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
