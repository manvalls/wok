package wok

import (
	"context"
	"net/http"
	"strings"
	"sync"
)

type Handler struct {
	Router
	App
	BasePath string
}

type LocalHandler struct {
	Handler
	*LocalRouter
	*LocalApp
}

func NewHandler(basePath string) LocalHandler {
	router := NewLocalRouter()
	app := NewLocalApp(router, basePath)

	return LocalHandler{
		LocalRouter: router,
		LocalApp:    app,
		Handler: Handler{
			Router:   router,
			App:      app,
			BasePath: basePath,
		},
	}
}

type emitter struct {
	listeners        map[string]map[*struct{}]func()
	mux              *sync.Mutex
	eventsByListener map[*struct{}]string
}

func newEmitter() emitter {
	return emitter{
		listeners:        map[string]map[*struct{}]func(){},
		eventsByListener: map[*struct{}]string{},
		mux:              &sync.Mutex{},
	}
}

func (e emitter) subscribe(event string, fn func()) (id *struct{}) {
	e.mux.Lock()
	defer e.mux.Unlock()

	m, ok := e.listeners[event]
	if !ok {
		m = map[*struct{}]func(){}
		e.listeners[event] = m
	}

	id = &struct{}{}
	m[id] = fn
	e.eventsByListener[id] = event
	return id
}

func (e emitter) unsubscribe(id *struct{}) {
	e.mux.Lock()
	defer e.mux.Unlock()

	event, ok := e.eventsByListener[id]
	if !ok {
		return
	}

	delete(e.eventsByListener, id)

	m, ok := e.listeners[event]
	if !ok {
		return
	}

	delete(m, id)
	if len(m) == 0 {
		delete(e.listeners, event)
	}
}

func (e emitter) emit(event string, id *struct{}) {
	e.mux.Lock()
	defer e.mux.Unlock()

	m, ok := e.listeners[event]
	if !ok {
		return
	}

	for _, fn := range m {
		go fn()
	}
}

func getControllerPlanKey(cp ControllerPlan) string {
	return cp.Method + "::" + cp.Controller + "?" + cp.Params.Encode()
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.BasePath) {
		return
	}

	url := r.URL
	url.Path = url.Path[len(h.BasePath):]

	triggerEmitter := newEmitter()
	flagsEmitter := newEmitter()
	flags := map[string]bool{}

	var result RouteResult
	var resultListeners []*struct{}
	var resultCancellers map[string]context.CancelFunc

	wg := sync.WaitGroup{}

	runControllerPlan := func(ctx context.Context, cp ControllerPlan) {
		defer wg.Done()
		// TODO: run the controller
	}

	resultMux := &sync.Mutex{}

	var updateRouteResult func()
	updateRouteResult = func() {
		resultMux.Lock()
		defer resultMux.Unlock()

		for _, id := range resultListeners {
			triggerEmitter.unsubscribe(id)
		}

		newResult := h.ResolveURL(r, url)
		newResultListeners := []*struct{}{}
		for _, trigger := range result.ReloadOn {
			id := triggerEmitter.subscribe(trigger, updateRouteResult)
			newResultListeners = append(newResultListeners, id)
		}

		newResultCancellers := map[string]context.CancelFunc{}

		for _, cp := range newResult.Controllers {
			key := getControllerPlanKey(cp)
			if cancel, ok := resultCancellers[key]; ok {
				newResultCancellers[key] = cancel
				delete(resultCancellers, key)
			} else {
				ctx, cancel := context.WithCancel(r.Context())
				newResultCancellers[key] = cancel

				wg.Add(1)
				go runControllerPlan(ctx, cp)
			}
		}

		for _, cancel := range resultCancellers {
			cancel()
		}

		result = newResult
		resultListeners = newResultListeners
		resultCancellers = newResultCancellers
	}

	result = h.ResolveURL(r, url)
	resultCancellers = map[string]context.CancelFunc{}

	resultListeners = []*struct{}{}
	for _, trigger := range result.ReloadOn {
		id := triggerEmitter.subscribe(trigger, updateRouteResult)
		resultListeners = append(resultListeners, id)
	}

	for _, cp := range result.Controllers {
		ctx, cancel := context.WithCancel(r.Context())
		resultCancellers[getControllerPlanKey(cp)] = cancel

		wg.Add(1)
		go runControllerPlan(ctx, cp)
	}

	wg.Wait()
	// TODO: send the response
}
