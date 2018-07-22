package wok

import (
	"context"
	"net/http"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/gddo/httputil"
	"github.com/manvalls/wit"
)

var head = wit.S("head")

// Scope wraps an HTTP request with useful internal data structures
type Scope struct {
	req         *http.Request
	usedHeaders map[string]bool
	routes      map[string]string
	mutex       sync.Mutex
}

// NewScope builds a new wok scope
func NewScope(req *http.Request) *Scope {
	return &Scope{req, map[string]bool{}, map[string]string{}, sync.Mutex{}}
}

// Write writes the specified delta to the specified ResponseWriter
func (s *Scope) Write(writer http.ResponseWriter, delta wit.Delta) (err error) {
	s.mutex.Lock()
	defer s.mutex.Unlock()

	script := "<script>(function(){var w=window.wok=window.wok||{};"
	for key, value := range s.routes {
		script += "w[" + strconv.Quote(key) + "]=" + strconv.Quote(value) + ";"
	}

	script += "})()</script>"

	delta = wit.List(delta, head.One(wit.Prepend(wit.FromString(script))))
	contentType := httputil.NegotiateContentType(s.req, []string{"text/html", "application/json"}, "text/html")

	var r wit.Renderer
	if contentType == "application/json" {
		r, err = wit.NewJSONRenderer(delta)
	} else {
		r, err = wit.NewHTMLRenderer(delta)
	}

	if err != nil {
		return err
	}

	resHeaders := writer.Header()
	for header := range s.usedHeaders {
		resHeaders["Vary"] = append(resHeaders["Vary"], header)
	}

	resHeaders["Vary"] = append(resHeaders["Vary"], "Content-Type")
	resHeaders["Vary"] = []string{strings.Join(resHeaders["Vary"], ", ")}
	resHeaders["Content-Type"] = []string{contentType}
	return r.Render(writer)
}

// FromHeader builds a route path from an HTTP header
func (s *Scope) FromHeader(header string) []uint {
	header = http.CanonicalHeaderKey(header)
	headerValue := strings.Join(s.req.Header[header], ",")

	route := make([]uint, 0)
	for _, h := range strings.Split(headerValue, ",") {
		n, err := strconv.ParseUint(strings.Trim(h, " "), 36, 64)
		if err == nil {
			route = append(route, uint(n))
		}
	}

	s.mutex.Lock()
	s.usedHeaders[header] = true
	s.mutex.Unlock()

	return route
}

// ToHeader maps a route path to a header value
func ToHeader(route ...uint) string {
	result := make([]string, len(route))
	for i, v := range route {
		result[i] = strconv.FormatUint(uint64(v), 36)
	}

	return strings.Join(result, ",")
}

// Run executes the given function under the context of the request
func (s *Scope) Run(f func(context.Context) wit.Delta) wit.Delta {
	return wit.Run(s.req.Context(), f)
}
