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
	offset    int
	params    Params
	oldParams Params
	command   wit.Command
}

func getOffset(previousRoute []string, newRoute []string) (offset int) {
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

func getDeps(request *Request) []string {
	request.deduperMutex.Lock()
	defer request.deduperMutex.Unlock()

	list := []string{}

	elem := request.firstDeduperElement
	for elem != nil {
		list = append(list, elem.key)
		elem = elem.next
	}

	return list
}

// Controller represents a controller of the routing tree
type Controller interface {
	Plan() Plan
	Resolve(string) Controller
}

// Default implements the default controller
type Default struct{}

// Plan returns the plan for this controller
func (controller Default) Plan() Plan {
	return Nil
}

// Resolve returns the nth child of this controller
func (controller Default) Resolve(id string) Controller {
	return Default{}
}

// HandleOptions wraps request.Handle options
type HandleOptions struct {
	Root       Controller
	HeaderName string
	Route      []string
	Params
}

// Handle executes the appropiate plans and gathers returned commands
func (r Request) Handle(o HandleOptions) (wit.Command, func()) {
	cond := sync.NewCond(&sync.Mutex{})
	cond.L.Lock()
	defer cond.L.Unlock()

	params := o.Params
	route := o.Route

	wg := sync.WaitGroup{}
	plansInfo := []*planInfo{}
	oldParams, oldRoute := r.FromHeader(o.HeaderName)
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
			controller := o.Root
			for i := 1; i < redirectionOffset; i++ {
				controller = controller.Resolve(route[i])
			}

			for i := redirectionOffset; i < len(route); i++ {
				if i != 0 {
					controller = controller.Resolve(route[i])
				}

				for _, c := range controller.Plan().Procedure().plans {
					if c.handler || c.deps != nil || i >= offset || paramsChanged(oldParams, params, c.params) {
						plansToRun = append(plansToRun, &planInfo{
							plan:   c,
							offset: i,
						})
					}
				}
			}
		}

		var redirectedRoute []string
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
				redirectionOffset = getOffset(route, redirectedRoute)
				route = redirectedRoute
			}
		}

	plansLoop:
		for _, info := range plansToRun {
			if redirectionHandled && info.offset >= redirectionOffset {
				continue
			}

			if r.IsSocket && info.socket == falseField {
				continue
			}

			if !(r.IsSocket && info.socket == trueField) {
				if r.IsNavigation {
					if !info.navigation {
						continue
					}
				} else {
					if !info.ajax {
						continue
					}
				}
			}

			if len(info.exclMethods) > 0 && info.exclMethods[r.Method] {
				continue
			}

			if len(info.methods) > 0 && !info.methods[r.Method] {
				continue
			}

			if len(info.exclCalls) > 0 && info.exclCalls[r.Call.Name] {
				continue
			}

			if len(info.calls) > 0 && !info.calls[r.Call.Name] {
				continue
			}

			if info.exclusive {
				for running > 0 {
					cond.Wait()

					r.customMutex.Lock()

					if *r.custom {
						r.customMutex.Unlock()
						break mainLoop
					}

					r.customMutex.Unlock()

					checkRedirections()

					if redirectionHandled && info.offset >= redirectionOffset {
						continue plansLoop
					}
				}
			}

			info.params = pickParams(params, info.plan.params)
			info.oldParams = pickParams(oldParams, info.plan.params)
			info.command = info.plan.command
			plansInfo = append(plansInfo, info)

			if info.fn != nil || info.doFn != nil {
				subRequest := r
				subRequest.Context, info.CancelFunc = context.WithCancel(r.Context)

				subRequest.redirectCond = cond
				subRequest.redirectedRoute = &redirectedRoute
				subRequest.redirectedParams = &redirectedParams

				subRequest.route = route
				subRequest.index = info.offset

				subRequest.Values = cloneParams(info.params)
				subRequest.OldParams = cloneParams(info.oldParams)

				if info.fn != nil {
					if info.sync {
						cond.L.Unlock()
						info.command = info.plan.fn(subRequest)
						cond.L.Lock()

						r.customMutex.Lock()

						if *r.custom {
							r.customMutex.Unlock()
							break mainLoop
						}

						r.customMutex.Unlock()

						checkRedirections()
					} else {
						running++

						go func(info *planInfo) {
							info.command = info.plan.fn(subRequest)

							cond.L.Lock()
							running--
							cond.Broadcast()
							cond.L.Unlock()
						}(info)
					}
				} else {
					if info.sync {
						info.plan.doFn(subRequest.ReadOnlyRequest)
					} else {
						wg.Add(1)
						go func(info *planInfo) {
							info.plan.doFn(subRequest.ReadOnlyRequest)
							wg.Done()
						}(info)
					}
				}
			}
		}

		for redirectedRoute == nil && redirectedParams == nil && running > 0 {
			cond.Wait()

			r.customMutex.Lock()

			if *r.custom {
				r.customMutex.Unlock()
				break mainLoop
			}

			r.customMutex.Unlock()

			checkRedirections()
		}

		r.customMutex.Lock()

		if *r.custom {
			r.customMutex.Unlock()
			break mainLoop
		}

		r.customMutex.Unlock()

		if redirectedRoute == nil && redirectedParams == nil {
			commandList := make([]wit.Command, len(plansInfo))
			depsList := getDeps(&r)

			for i, info := range plansInfo {
				if info.deps != nil {
					depsCommands := []wit.Command{}
					for _, dep := range depsList {
						depsCommands = append(depsCommands, info.deps(dep))
					}

					commandList[i] = wit.List(depsCommands...)
				} else if info.doFn == nil {
					commandList[i] = info.command
				}
			}

			key := &struct{}{}

			r.routesMutex.Lock()
			r.routes[key] = headerAndValue{o.HeaderName, ToHeader(params, route...)}
			r.routesMutex.Unlock()

			go func() {
				<-r.Done()
				r.routesMutex.Lock()
				delete(r.routes, key)
				r.routesMutex.Unlock()
			}()

			r.ContextVary(o.HeaderName)
			return wit.List(commandList...), func() {
				wg.Wait()
			}
		}
	}

	return wit.Nil, func() {
		wg.Wait()
	}
}
