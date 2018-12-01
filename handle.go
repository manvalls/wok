package wok

import (
	"context"
	"sync"

	"github.com/manvalls/wit"
)

const maxRedirections = 1000

type planInfo struct {
	plan
	context.CancelFunc
	offset int
	params Params
	action wit.Action
}

func getOffset(previousRoute []uint, newRoute []uint) (offset int) {
	offset = len(previousRoute)
	for i, j := range previousRoute {
		if i >= len(newRoute) || newRoute[i] != j {
			offset = i
			return
		}
	}

	return
}

func paramsMatch(params Params, usedParams Params) bool {
	paramsList := make([]string, len(usedParams))

	i := 0
	for key := range usedParams {
		paramsList[i] = key
		i++
	}

	return paramsChanged(params, usedParams, paramsList)
}

func pickParams(params Params, paramsList []string) Params {
	return params
}

func paramsChanged(oldParams Params, newParams Params, paramsList []string) bool {
	for _, param := range paramsList {
		newParam := newParams[param]
		oldParam := oldParams[param]

		if len(newParam) != len(oldParam) {
			return true
		}

		for i := range newParam {
			if newParam[i] != oldParam[i] {
				return true
			}
		}
	}

	return false
}

func cloneParams(params Params) Params {
	result := make(Params)
	for key, value := range params {
		clone := make([]string, len(value))
		copy(clone, value)
		result[key] = clone
	}

	return result
}

// Controller represents a controller of the routing tree
type Controller interface {
	Plan() Plan
	Resolve(uint) Controller
}

// Default implements the default controller
type Default struct{}

// Plan returns the plan for this controller
func (controller Default) Plan() Plan {
	return Nil
}

// Resolve returns the nth child of this controller
func (controller Default) Resolve(id uint) Controller {
	return Default{}
}

// Handle executes the appropiate plans and gathers returned actions
func (r Request) Handle(rootController Controller, header string, params Params, route ...uint) wit.Action {
	cond := sync.NewCond(&sync.Mutex{})
	cond.L.Lock()
	defer cond.L.Unlock()

	plansInfo := []*planInfo{}
	oldParams, oldRoute := r.FromHeader(header)
	redirectionOffset := 0
	running := 0

mainLoop:
	for i := 0; i <= maxRedirections; i++ {
		plansToRun := []*planInfo{}
		oldPlansInfo := plansInfo
		plansInfo = []*planInfo{}

		offset := getOffset(oldRoute, route)
		minOffset := offset
		if redirectionOffset < minOffset {
			minOffset = redirectionOffset
		}

		for _, info := range oldPlansInfo {
			if info.offset >= minOffset {
				if info.fn != nil {
					info.CancelFunc()
				}

				continue
			}

			if !paramsMatch(params, info.params) {
				if info.fn != nil {
					info.CancelFunc()
				}

				if info.handler || paramsChanged(oldParams, params, info.plan.params) {
					plansToRun = append(plansToRun, info)
				}
				continue
			}

			plansInfo = append(plansInfo, info)
		}

		if redirectionOffset < len(route) {
			controller := rootController
			for i := 0; i < redirectionOffset; i++ {
				controller = controller.Resolve(route[i])
			}

			for i := redirectionOffset; i < len(route); i++ {
				controller = controller.Resolve(route[i])
				for _, c := range controller.Plan().Procedure().plans {
					if c.handler || i >= offset || paramsChanged(oldParams, params, c.params) {
						plansToRun = append(plansToRun, &planInfo{
							plan:   c,
							offset: i,
						})
					}
				}
			}
		}

		var redirectedRoute []uint
		var redirectedParams Params
		redirectionHandled := false

		checkRedirections := func() {
			if redirectionHandled || (redirectedParams == nil && redirectedRoute == nil) {
				return
			}

			redirectionHandled = true
			if redirectedParams != nil {
				params = redirectedParams
			}

			if redirectedRoute != nil {
				redirectionHandled = true
				redirectionOffset = getOffset(route, redirectedRoute)
				route = redirectedRoute
			}
		}

		for _, info := range plansToRun {
			if redirectionHandled && info.offset >= redirectionOffset {
				continue
			}

			if info.exclusive {
				for running > 0 {
					cond.Wait()

					r.customBodyMutex.Lock()

					if *r.customBody {
						r.customBodyMutex.Unlock()
						break mainLoop
					}

					r.customBodyMutex.Unlock()

					checkRedirections()

					if redirectionHandled && info.offset >= redirectionOffset {
						continue
					}
				}
			}

			info.params = pickParams(params, info.plan.params)
			info.action = info.plan.action
			plansInfo = append(plansInfo, info)

			if info.fn != nil {
				subRequest := r
				subRequest.Context, info.CancelFunc = context.WithCancel(r.Context)

				subRequest.redirectCond = cond
				subRequest.redirectedRoute = &redirectedRoute
				subRequest.redirectedParams = &redirectedParams

				subRequest.route = route
				subRequest.index = info.offset

				subRequest.Values = cloneParams(info.params)

				if info.async {
					running++

					go func(info *planInfo) {
						info.action = info.plan.fn(subRequest)

						cond.L.Lock()
						running--
						cond.Broadcast()
						cond.L.Unlock()
					}(info)
				} else {
					cond.L.Unlock()
					info.action = info.plan.fn(subRequest)
					cond.L.Lock()

					r.customBodyMutex.Lock()

					if *r.customBody {
						r.customBodyMutex.Unlock()
						break mainLoop
					}

					r.customBodyMutex.Unlock()

					checkRedirections()
				}
			}
		}

		for redirectedRoute == nil && redirectedParams == nil && running > 0 {
			cond.Wait()

			r.customBodyMutex.Lock()

			if *r.customBody {
				r.customBodyMutex.Unlock()
				break mainLoop
			}

			r.customBodyMutex.Unlock()

			checkRedirections()
		}

		r.customBodyMutex.Lock()

		if *r.customBody {
			r.customBodyMutex.Unlock()
			break mainLoop
		}

		r.customBodyMutex.Unlock()

		if redirectedRoute == nil && redirectedParams == nil {
			actionList := make([]wit.Action, len(plansInfo))
			for i, info := range plansInfo {
				actionList[i] = info.action
			}

			key := &struct{}{}

			r.routesMutex.Lock()
			r.routes[key] = headerAndValue{header, ToHeader(params, route...)}
			r.routesMutex.Unlock()

			go func() {
				<-r.Done()
				r.routesMutex.Lock()
				delete(r.routes, key)
				r.routesMutex.Unlock()
			}()

			r.ContextVary(header)
			return wit.List(actionList...)
		}
	}

	return wit.Nil
}
