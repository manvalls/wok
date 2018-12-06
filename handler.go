package wok

import (
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/gddo/httputil"
	"github.com/manvalls/way"
	"github.com/manvalls/wit"
)

var toRemove = wit.S("[data-wok-remove]")

// Handler implements an HTTP fn which provides wok requests
type Handler struct {
	Root        func() Controller
	RouteHeader string
	DepsHeader  string
	way.Router
}

func (h Handler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	params, route, err := h.GetRoute(r.URL)
	if err != nil {
		w.WriteHeader(404)
		return
	}

	routeHeader := h.RouteHeader
	if routeHeader == "" {
		routeHeader = "X-Wok-Route"
	}

	depsHeader := h.DepsHeader
	if depsHeader == "" {
		depsHeader = "X-Wok-Deps"
	}

	var customBodyReader io.Reader
	customBody := false

	request := Request{
		Request:        r,
		ResponseWriter: w,
		Context:        r.Context(),
		Router:         h.Router,

		RequestHeader:  r.Header,
		ResponseHeader: w.Header(),

		StatusCodeGetterSetter: &StatusCodeGetterSetter{},

		customBody:       &customBody,
		customBodyReader: &customBodyReader,
		customBodyMutex:  &sync.Mutex{},

		routes:      make(map[*struct{}]headerAndValue),
		routesMutex: &sync.Mutex{},

		vary:      make(map[string]int),
		varyMutex: &sync.Mutex{},

		deduper: &deduper{
			indexedDeduperElements: make(map[string]*deduperElement),
			deduperMutex:           &sync.Mutex{},
		},
	}

	_, deps := request.FromHeader(depsHeader)
	depsMap := map[string]bool{}
	for _, dep := range deps {
		depsMap[dep] = true
	}

	request.loadedDependencies = depsMap

	delta := request.Handle(h.Root(), routeHeader, params, route...)
	usedDeps := getDeps(&request)

	if len(usedDeps) != 0 {
		request.Vary(depsHeader)
		script := "<script data-wok-remove>!function(){var n,o='" +
			depsHeader +
			"',i=window.SPH=window.SPH||{},p=i.deps=i.deps||{},s={},t=[],w=(p[o]?p[o].split(','):[]).concat('" +
			ToHeader(nil, usedDeps...) +
			"'.split(','));for(n=0;n<w.length;n++)s.hasOwnProperty(w[n])||(s[w[n]]=1,t.push(w[n]));p[o]=t.join(',')}();</script>"

		delta = wit.List(delta, wit.Head.One(wit.Prepend(wit.FromString(script))))
	}

	request.varyMutex.Lock()
	defer request.varyMutex.Unlock()

	resHeaders := w.Header()
	for header, n := range request.vary {
		if n > 0 {
			resHeaders["Vary"] = append(resHeaders["Vary"], header)
		}
	}

	request.customBodyMutex.Lock()
	defer request.customBodyMutex.Unlock()

	if !customBody {
		request.routesMutex.Lock()
		defer request.routesMutex.Unlock()

		routes := map[string]string{}
		for _, hv := range request.routes {
			routes[hv.header] = hv.value
		}

		script := "<script data-wok-remove>(function(){var w=window.SPH=window.SPH||{};w.routes={"

		i := 0
		for key, value := range routes {
			if i != 0 {
				script += ","
			}

			script += strconv.Quote(key) + ":" + strconv.Quote(value)
			i++
		}

		script += "}})()</script>"

		delta = wit.List(toRemove.All(wit.Remove), delta, wit.Head.One(wit.Prepend(wit.FromString(script))))
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
		w.WriteHeader(request.StatusCode())
		if customBodyReader != nil {
			io.Copy(w, customBodyReader)
		}
	}
}
