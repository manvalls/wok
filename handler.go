package wok

import (
	"math/rand"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/golang/gddo/httputil"
	"github.com/gorilla/websocket"
	"github.com/manvalls/way"
	"github.com/manvalls/wit"
	"github.com/oklog/ulid"
)

// Handler implements an HTTP handler which provides wok requests
type Handler struct {
	Root             func() Controller
	RootName         string
	RouteHeader      string
	DepsHeader       string
	InstanceIDHeader string
	InputBuffer      int
	websocket.Upgrader
	way.Router
}

// CallData holds a call's data
type CallData struct {
	Name string
	url.Values
}

var randPool = sync.Pool{
	New: func() interface{} {
		return rand.New(
			rand.NewSource(
				time.Now().UnixNano(),
			),
		)
	},
}

func (h Handler) serve(w http.ResponseWriter, r *http.Request, input <-chan url.Values, output chan<- wit.Command, flush func()) {
	params, route, err := h.GetRoute(r.URL)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	route = append([]string{h.RootName}, route...)

	routeHeader := h.RouteHeader
	if routeHeader == "" {
		routeHeader = "X-Wok-Route"
	}

	depsHeader := h.DepsHeader
	if depsHeader == "" {
		depsHeader = "X-Wok-Deps"
	}

	instanceIDHeader := h.InstanceIDHeader
	if instanceIDHeader == "" {
		instanceIDHeader = "X-Wok-Instance-ID"
	}

	var instanceCmd wit.Command
	instanceID := r.Header.Get(instanceIDHeader)
	if instanceID == "" {
		r := randPool.Get().(*rand.Rand)

		u, err := ulid.New(
			ulid.Timestamp(time.Now()),
			r,
		)

		randPool.Put(r)

		if err != nil {
			instanceID = ""
		} else {
			instanceID = u.String()
			script := "<script data-w-rm>!function(){(window.SPASessionHeadersMap=window.SPH=window.SPASessionHeadersMap||window.SPH||{}).instance={'" + instanceIDHeader + "':'" + instanceID + "'};}()</script>"
			instanceCmd = wit.Head.One(wit.Append(wit.FromString(script)))
		}
	}

	var customHandler func(http.ResponseWriter)
	custom := false

	request := Request{
		w:              w,
		ResponseHeader: w.Header(),

		ReadOnlyRequest: ReadOnlyRequest{
			InitialLoad:   r.Header.Get("X-Requested-With") == "",
			IsSocket:      input != nil && output != nil && flush != nil,
			IsNavigation:  r.Header.Get("X-Requested-With") != "XMLHttpRequest" || r.Header.Get("X-Navigation") == "true",
			InstanceID:    instanceID,
			RequestHeader: r.Header,
			Router:        h.Router,
			Request:       r,
			Context:       r.Context(),
			Input:         input,
			Output:        output,
			Mutex:         &sync.Mutex{},
		},

		StatusCodeGetterSetter: &StatusCodeGetterSetter{},

		custom:        &custom,
		customHandler: &customHandler,
		customMutex:   &sync.Mutex{},

		routes:      make(map[*struct{}]headerAndValue),
		routesMutex: &sync.Mutex{},

		vary:      make(map[string]int),
		varyMutex: &sync.Mutex{},

		deduper: &deduper{
			indexedDeduperElements: make(map[string]*deduperElement),
			deduperMutex:           &sync.Mutex{},
		},
	}

	callParts := strings.SplitN(r.Header.Get("X-Wok-Call"), "?", 2)

	switch len(callParts) {
	case 2:
		request.Call.Values, _ = url.ParseQuery(callParts[1])
		fallthrough
	case 1:
		request.Call.Name = callParts[0]
	case 0:
	}

	_, deps := request.FromHeader(depsHeader)
	depsMap := map[string]bool{}
	for _, dep := range deps {
		depsMap[dep] = true
	}

	request.loadedDependencies = depsMap

	delta, doWait := request.Handle(HandleOptions{
		Root:       h.Root(),
		HeaderName: routeHeader,
		Params:     params,
		Route:      route,
	})

	if instanceCmd != nil {
		delta = wit.List(delta, instanceCmd)
	}

	usedDeps := getDeps(&request)

	if len(usedDeps) != 0 {
		request.Vary(depsHeader)

		list := ""
		for i, dep := range usedDeps {
			if i != 0 {
				list += ","
			}

			list += strconv.Quote(dep)
		}

		script := "<script data-w-rm>!function(){var w=window,o='" + depsHeader + "',i=w.SPASessionHeadersMap=w.SPH=w.SPH||w.window.SPASessionHeadersMap||{},p=i.deps=i.deps||{};(p[o]=p[o]||[]).push(" + list + ");}()</script>"
		delta = wit.List(delta, wit.Head.One(wit.Append(wit.FromString(script))))
	}

	request.varyMutex.Lock()
	defer request.varyMutex.Unlock()

	resHeaders := w.Header()
	for header, n := range request.vary {
		if n > 0 {
			resHeaders["Vary"] = append(resHeaders["Vary"], header)
		}
	}

	request.customMutex.Lock()
	defer request.customMutex.Unlock()

	if !custom {
		request.routesMutex.Lock()
		defer request.routesMutex.Unlock()

		routes := map[string]string{}
		for _, hv := range request.routes {
			routes[hv.header] = hv.value
		}

		if request.IsNavigation {
			script := "<script data-w-rm>(function(){(window.SPH=window.SPASessionHeadersMap=window.SPH||window.SPASessionHeadersMap||{}).routes={"

			i := 0
			for key, value := range routes {
				if i != 0 {
					script += ","
				}

				script += strconv.Quote(key) + ":" + strconv.Quote(value)
				i++
			}

			script += "}})()</script>"

			delta = wit.List(delta, wit.Head.One(wit.Append(wit.FromString(script))))
		}

		contentType := httputil.NegotiateContentType(r, []string{"text/html", "application/json"}, "text/html")

		var renderer wit.Renderer
		if contentType == "application/json" {
			renderer = wit.NewJSONRenderer(delta)
		} else {
			contentType += "; charset=utf-8"
			renderer = wit.NewHTMLRenderer(delta)
		}

		resHeaders["Vary"] = append(resHeaders["Vary"], "Accept")
		resHeaders["Vary"] = []string{strings.Join(resHeaders["Vary"], ", ")}

		resHeaders["Content-Type"] = []string{contentType}
		w.WriteHeader(request.StatusCode())
		renderer.Render(w)
	} else {
		resHeaders["Vary"] = []string{strings.Join(resHeaders["Vary"], ", ")}
		if customHandler != nil {
			customHandler(w)
		} else {
			w.WriteHeader(request.StatusCode())
		}
	}

	if flush != nil {
		flush()
	}

	doWait()
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	if strings.ToLower(r.Header.Get("Upgrade")) == "websocket" {
		conn, err := h.Upgrade(w, r, nil)
		if err == nil {
			h.handleWS(r.Context(), conn)
		}
	} else {
		h.serve(w, r, nil, nil, nil)
	}
}
