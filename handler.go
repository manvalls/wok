package wok

import (
	"context"
	"net/http"
	"net/url"
	"strings"
	"sync"

	"github.com/manvalls/wit"
)

type Handler struct {
	Router
	App

	BasePath string
	Host     string
	FrameId  func(*http.Request) string
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

type controllerProcess struct {
	ctx            context.Context
	cancel         context.CancelFunc
	controllerPlan ControllerPlan

	eventsSinceStart map[string]bool
	eventsListenedTo map[string]bool
	eventsMux        *sync.Mutex
}

func newControllerProcess(ctx context.Context, cp ControllerPlan) *controllerProcess {
	childCtx, cancel := context.WithCancel(ctx)
	return &controllerProcess{
		ctx:            childCtx,
		cancel:         cancel,
		controllerPlan: cp,

		eventsSinceStart: map[string]bool{},
		eventsListenedTo: map[string]bool{},
		eventsMux:        &sync.Mutex{},
	}
}

func getControllerPlanKey(cp ControllerPlan) string {
	return cp.Method + "::" + cp.Controller + "?" + cp.Params.Encode()
}

func (h Handler) Attach(m *http.ServeMux) {
	m.Handle(strings.TrimRight(h.BasePath, "/")+"/", h)
}

func (h Handler) Compute(url *url.URL, r *http.Request) (wit.Delta, int, string, http.Header) {
	parentContext := r.Context()

	var result RouteResult
	var controllerProcesses map[string]*controllerProcess

	statusCode := 200
	location := ""
	headers := http.Header{}

	wg := &sync.WaitGroup{}

	var runControllerPlan func(p *controllerProcess)

	updateRouteResult := func() {
		newResult := h.ResolveURL(r, url)
		newControllerProcesses := map[string]*controllerProcess{}

		for _, cp := range newResult.Controllers {
			key := getControllerPlanKey(cp)
			if p, ok := controllerProcesses[key]; ok {
				newControllerProcesses[key] = p
				delete(controllerProcesses, key)
			} else {
				p := newControllerProcess(parentContext, cp)
				newControllerProcesses[key] = p

				wg.Add(1)
				go runControllerPlan(p)
			}
		}

		for _, p := range controllerProcesses {
			p.cancel()
		}

		result = newResult
		controllerProcesses = newControllerProcesses
	}

	mux := sync.Mutex{}

	trigger := func(event string) {
		mux.Lock()
		defer mux.Unlock()

		needsResultUpdate := false
		for _, ro := range result.ReloadOn {
			if ro == event {
				needsResultUpdate = true
				break
			}
		}

		if needsResultUpdate {
			updateRouteResult()
		}

		for key, p := range controllerProcesses {
			p.eventsMux.Lock()

			if p.eventsListenedTo[event] {
				p.cancel()
				np := newControllerProcess(parentContext, p.controllerPlan)
				controllerProcesses[key] = np

				wg.Add(1)
				go runControllerPlan(np)
			} else {
				p.eventsSinceStart[event] = true
			}

			p.eventsMux.Unlock()
		}

	}

	runControllerPlan = func(p *controllerProcess) {
		defer wg.Done()
		// TODO: run the controller
	}

	result = h.ResolveURL(r, url)
	controllerProcesses = map[string]*controllerProcess{}

	mux.Lock()

	for _, cp := range result.Controllers {
		p := newControllerProcess(parentContext, cp)
		controllerProcesses[getControllerPlanKey(cp)] = p

		wg.Add(1)
		go runControllerPlan(p)
	}

	mux.Unlock()

	wg.Wait()
	return nil, statusCode, location, headers
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if !strings.HasPrefix(r.URL.Path, h.BasePath) {
		return
	}

	doc := wit.NewDocument()

	url := *r.URL
	url.Path = url.Path[len(h.BasePath):]

	delta, statusCode, location, header := h.Compute(&url, r)

	wh := w.Header()
	for key, value := range header {
		wh[key] = value
	}

	if delta == nil {
		w.Header().Add("location", location)
		w.WriteHeader(statusCode)
		return
	}

	// TODO: send JSON if requested

	delta.Apply(doc)
	w.Header().Add("content-type", "text/html; charset=utf-8")
	w.WriteHeader(statusCode)
	doc.Render(w)
}
