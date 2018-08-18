package wok

import (
	"context"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"

	"github.com/golang/gddo/httputil"
	"github.com/manvalls/wit"
)

var toRemove = wit.S("[data-wok-remove]")

// Params hold information about route parameters
type Params = map[string][]string

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

	script := "<script data-wok-remove>(function(){var w=window.wok=window.wok||{};w.routes={"

	i := 0
	for key, value := range s.routes {
		if i != 0 {
			script += ","
		}

		script += strconv.Quote(key) + ":" + strconv.Quote(value)
		i++
	}

	script += "}})()</script>"

	delta = wit.List(toRemove.All(wit.Remove), delta, wit.Head.One(wit.Prepend(wit.FromString(script))))
	contentType := httputil.NegotiateContentType(s.req, []string{"text/html", "application/json"}, "text/html")

	var r wit.Renderer
	if contentType == "application/json" {
		r, err = wit.NewJSONRenderer(delta)
	} else {
		contentType += "; charset=utf-8"
		r, err = wit.NewHTMLRenderer(delta)
	}

	if err != nil && err != wit.ErrEnd {
		return err
	}

	resHeaders := writer.Header()
	for header := range s.usedHeaders {
		resHeaders["Vary"] = append(resHeaders["Vary"], header)
	}

	resHeaders["Vary"] = append(resHeaders["Vary"], "Accept")
	resHeaders["Vary"] = []string{strings.Join(resHeaders["Vary"], ", ")}

	if err == nil {
		resHeaders["Content-Type"] = []string{contentType}
		return r.Render(writer)
	}

	return nil
}

// FromHeader builds a route path from an HTTP header
func (s *Scope) FromHeader(header string) (Params, []uint) {
	header = http.CanonicalHeaderKey(header)
	headerValue := strings.Join(s.req.Header[header], ",")

	rawRoute := ""
	rawQuery := ""

	parts := strings.Split(headerValue, "?")

	switch len(parts) {
	case 0:
	case 1:
		rawRoute = parts[0]
	default:
		rawRoute = parts[0]
		rawQuery = parts[1]
	}

	route := make([]uint, 0)
	for _, h := range strings.Split(rawRoute, ",") {
		n, err := strconv.ParseUint(strings.Trim(h, " "), 36, 64)
		if err == nil {
			route = append(route, uint(n))
		}
	}

	query, err := url.ParseQuery(rawQuery)
	if err != nil {
		query = url.Values{}
	}

	s.mutex.Lock()
	s.usedHeaders[header] = true
	s.mutex.Unlock()

	return query, route
}

// ToHeader maps a route path to a header value
func ToHeader(params Params, route ...uint) string {
	result := make([]string, len(route))
	for i, v := range route {
		result[i] = strconv.FormatUint(uint64(v), 36)
	}

	var values url.Values
	path := strings.Join(result, ",")
	values = params
	query := values.Encode()

	if query != "" {
		return path + "?" + query
	}

	return path
}

// Run executes the given function under the context of the request
func (s *Scope) Run(f func(context.Context) wit.Delta) wit.Delta {
	return wit.Run(s.req.Context(), f)
}
