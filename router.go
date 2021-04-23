package wok

import (
	"errors"
	"net/http"
	"net/url"
	"strings"
)

type pathNode struct {
	children    map[string]*pathNode
	suffix      *pathNode
	route       string
	parameters  []string
	extraParams Params
}

type routeMappingNode struct {
	children    map[string]*routeMappingNode
	parts       []*pathPart
	extraParams ExtraParams
}

type routeMapping struct {
	usedParams map[string]int
	root       *routeMappingNode
}

type RouteRedirection struct {
	Route string
	Params
	ReloadOn []string
}

type MapFunc = func(r *http.Request, route string, params Params) RouteRedirection

type RouteHandlerFunc = func(r *http.Request, route string, params Params) RouteResult

type RouteController struct {
	Controller string
	Method     string
	Params     []string

	DependsOn       []string
	RunAfter        []string
	Persistent      bool
	Batch           bool
	Lazy            bool
	Socket          bool
	NeedsCleanup    bool
	NeedsValidation bool
	Cache           bool
	Prefetch        bool
}

type LocalRouter struct {
	pathRoot      *pathNode
	mapFuncs      map[string][]*MapFunc
	routeMappings map[string]*routeMapping
	handlers      map[string][]RouteHandlerFunc
}

func NewLocalRouter() *LocalRouter {
	return &LocalRouter{
		pathRoot:      &pathNode{children: map[string]*pathNode{}},
		mapFuncs:      map[string][]*MapFunc{},
		routeMappings: map[string]*routeMapping{},
		handlers:      map[string][]RouteHandlerFunc{},
	}
}

type pathPart struct {
	part     string
	isParam  bool
	isSuffix bool
}

type RoutePaths = map[string]ExtraParams
type Routes = map[string]RoutePaths
type ExtraParams = map[string]string

func RoutePath(path string) RoutePaths {
	return RoutePaths{
		path: ExtraParams{},
	}
}

func getMappingKey(params Params, mapping *routeMapping) [][]string {
	mappingKey := make([][]string, len(mapping.usedParams))
	largestIndex := 0

	for key, i := range mapping.usedParams {
		if len(params[key]) > 0 {
			mappingKey[i] = make([]string, len(params[key]))
			for j, p := range params[key] {
				mappingKey[i][j] = "&" + url.QueryEscape(p)
			}

			if i > largestIndex {
				largestIndex = i
			}
		} else {
			mappingKey[i] = []string{""}
		}
	}

	if largestIndex >= len(mappingKey)-1 {
		return mappingKey
	}

	return mappingKey[:largestIndex+1]
}

func fillRouteMapping(node *routeMappingNode, keypath [][]string, extraParams ExtraParams, parts []*pathPart) {
	if len(keypath) == 0 {
		node.parts = parts
		node.extraParams = extraParams
		return
	}

	key := keypath[0]
	keypath = keypath[1:]

	child := &routeMappingNode{
		children: map[string]*routeMappingNode{},
	}

	fillRouteMapping(child, keypath, extraParams, parts)
	node.children[key[0]] = child
}

func findRouteParts(keypath [][]string, node *routeMappingNode) (parts []*pathPart, extraParams ExtraParams, ok bool) {
	if len(keypath) == 0 {
		return node.parts, node.extraParams, node.parts != nil
	}

	key := keypath[0]
	keypath = keypath[1:]

	for _, subkey := range key {
		child, ok := node.children[subkey]
		if ok {
			parts, extraParams, ok := findRouteParts(keypath, child)
			if ok {
				return parts, extraParams, ok
			}
		}
	}

	if len(key) != 1 || key[0] != "" {
		child, ok := node.children[""]
		if ok {
			parts, extraParams, ok := findRouteParts(keypath, child)
			if ok {
				return parts, extraParams, ok
			}
		}
	}

	return nil, nil, false
}

func extraParamsToParams(extraParams ExtraParams) Params {
	params := Params{}
	for key, value := range extraParams {
		params[key] = []string{value}
	}

	return params
}

func (r *LocalRouter) addRoute(route string, path string, extraParams ExtraParams) {
	pathParent := r.pathRoot
	parts := []*pathPart{}
	currentPart := ""
	currentParam := ""
	isParam := false
	params := []string{}
	hasSuffix := false

	flush := func() {
		next := ""
		isSuffix := false

		if isParam {
			if currentParam == "" {
				return
			}

			lastPos := len(currentParam) - 1
			if currentParam[lastPos] == '*' {
				currentParam = currentParam[:lastPos]
				isSuffix = true
				hasSuffix = true
			}

			decodedParam, err := url.QueryUnescape(currentParam)
			if err == nil {
				currentParam = decodedParam
			}

			parts = append(parts, &pathPart{currentParam, true, isSuffix})
			params = append(params, currentParam)
		} else {
			if currentPart == "" {
				return
			}

			decodedPart, err := url.QueryUnescape(currentPart)
			if err == nil {
				currentPart = decodedPart
			}

			parts = append(parts, &pathPart{currentPart, false, false})
			next = currentPart
		}

		if !isSuffix {
			nextParent := pathParent.children[next]
			if nextParent == nil {
				nextParent = &pathNode{children: map[string]*pathNode{}}
				pathParent.children[next] = nextParent
			}
			pathParent = nextParent
		} else {
			nextParent := pathParent.suffix
			if nextParent == nil {
				nextParent = &pathNode{children: map[string]*pathNode{}}
				pathParent.suffix = nextParent
			}
			pathParent = nextParent
		}

		currentParam = ""
		currentPart = ""
		isParam = false
		return
	}

	for _, c := range path {
		if hasSuffix {
			break
		}

		switch c {
		case '/':
			flush()

		case ':':
			if currentParam == "" && currentPart == "" {
				isParam = true
			} else {
				if isParam {
					currentParam += string(c)
				} else {
					currentPart += string(c)
				}
			}
		case '?':
			flush()
			break
		default:
			if isParam {
				currentParam += string(c)
			} else {
				currentPart += string(c)
			}
		}
	}

	flush()

	pathParent.route = route
	pathParent.parameters = params
	pathParent.extraParams = extraParamsToParams(extraParams)

	mapping, ok := r.routeMappings[route]
	if !ok {
		mapping = &routeMapping{
			usedParams: map[string]int{},
			root: &routeMappingNode{
				children: map[string]*routeMappingNode{},
			},
		}

		r.routeMappings[route] = mapping
	}

	for key := range extraParams {
		if _, ok = mapping.usedParams[key]; !ok {
			i := len(mapping.usedParams)
			mapping.usedParams[key] = i
		}
	}

	node := mapping.root
	keypath := getMappingKey(pathParent.extraParams, mapping)
	fillRouteMapping(node, keypath, extraParams, parts)
}

func (r *LocalRouter) AddRoute(route string, paths RoutePaths) {
	for path, params := range paths {
		r.addRoute(route, path, params)
	}
}

func (r *LocalRouter) AddRoutes(routes Routes) {
	for route, paths := range routes {
		r.AddRoute(route, paths)
	}
}

func Merge(params ...Params) Params {
	result := make(Params)

	for _, p := range params {
		for paramName, paramValues := range p {
			for _, value := range paramValues {
				result[paramName] = append(result[paramName], value)
			}
		}
	}

	return result
}

var errNotFound = errors.New("Requested route not found")

func match(parts []string, params []string, parent *pathNode) ([]string, *pathNode, error) {
	if len(parts) == 0 {
		if parent.route == "" {
			return nil, nil, errNotFound
		}

		return params, parent, nil
	}

	keys := []string{parts[0], ""}
	nextParts := parts[1:]

	for _, key := range keys {
		child := parent.children[key]
		if child != nil {
			nextParams := params
			if key == "" {
				nextParams = append(params, parts[0])
			}

			p, m, err := match(nextParts, nextParams, child)
			if err == nil {
				return p, m, nil
			}
		}
	}

	if parent.suffix != nil {
		suffix := ""
		for i, part := range parts {
			if i != 0 {
				suffix += "/"
			}

			suffix += url.QueryEscape(part)
		}

		nextParams := append(params, suffix)
		return match([]string{}, nextParams, parent.suffix)
	}

	return nil, nil, errNotFound
}

func (r *LocalRouter) Map(route string, mapFn MapFunc) {
	r.mapFuncs[route] = append(r.mapFuncs[route], &mapFn)
}

func (r *LocalRouter) ResolveURL(req *http.Request, urlToMatch *url.URL) RouteResult {
	currentPart := ""
	parts := []string{}
	path := urlToMatch.Path

	flush := func() {
		if currentPart == "" {
			return
		}

		decodedPart, err := url.QueryUnescape(currentPart)
		if err == nil {
			currentPart = decodedPart
		}

		parts = append(parts, currentPart)
		currentPart = ""
		return
	}

	for _, c := range path {
		switch c {
		case '/':
			flush()

		default:
			currentPart += string(c)
		}
	}

	flush()

	paramList, matchedPart, err := match(parts, []string{}, r.pathRoot)
	if err != nil {
		return RouteResult{}
	}

	params := make(Params)
	for i, p := range matchedPart.parameters {
		params[p] = append(params[p], paramList[i])
	}

	resolvedParams := Merge(matchedPart.extraParams, params, urlToMatch.Query())
	finalRoute, finalParams, reloadOn := r.runMaps(req, matchedPart.route, resolvedParams)
	return r.resolve(req, finalRoute, finalParams, reloadOn)
}

func (r *LocalRouter) runMaps(req *http.Request, route string, params Params) (currentRoute string, currentParams Params, reloadOn []string) {
	reloadOn = []string{}
	currentRoute = route
	currentParams = params
	ranFuncs := map[*MapFunc]bool{}

	for {
		var mapFunc *MapFunc
		prefix := currentRoute
		splitRoute := strings.Split(currentRoute, ".")

	loop:
		for len(splitRoute) > 0 {
			mapFuncs, ok := r.mapFuncs[prefix]
			if ok {
				for _, fn := range mapFuncs {
					if !ranFuncs[fn] {
						mapFunc = fn
						break loop
					}
				}
			}

			splitRoute = splitRoute[:len(splitRoute)-1]
			prefix = strings.Join(splitRoute, ".")
		}

		if mapFunc == nil {
			return
		}

		ranFuncs[mapFunc] = true
		redirection := (*mapFunc)(req, currentRoute, currentParams)
		currentRoute = redirection.Route
		currentParams = redirection.Params
		reloadOn = append(reloadOn, redirection.ReloadOn...)
	}
}

func (r *LocalRouter) ResolveRoute(req *http.Request, route string, params Params) (resolvedURL string, result RouteResult) {
	mapping, ok := r.routeMappings[route]
	if !ok {
		return "", RouteResult{}
	}

	keypath := getMappingKey(params, mapping)

	parts, extraParams, ok := findRouteParts(keypath, mapping.root)
	if !ok {
		return "", RouteResult{}
	}

	pathResult := ""

	queryParams := Merge(params)

	if len(parts) == 0 {
		pathResult = "/"
	} else {

		for _, part := range parts {
			if part.isSuffix {
				param, ok := queryParams[part.part]
				if ok && len(param) > 0 {
					pathResult += "/" + param[0]
					if len(queryParams[part.part]) > 1 {
						queryParams[part.part] = queryParams[part.part][1:]
					} else {
						delete(queryParams, part.part)
					}
				}
			} else if part.isParam {
				pathResult += "/"
				param, ok := queryParams[part.part]
				if ok && len(param) > 0 {
					pathResult += url.QueryEscape(param[0])
					if len(queryParams[part.part]) > 1 {
						queryParams[part.part] = queryParams[part.part][1:]
					} else {
						delete(queryParams, part.part)
					}
				}
			} else {
				pathResult += "/" + part.part
			}
		}

	}

	query := Params{}
	for key, values := range queryParams {
		if v, ok := extraParams[key]; ok {
			if len(values) > 1 {
				query[key] = make([]string, 0, len(values)-1)

				var i int
				var value string

				for i, value = range values {
					if value == v {
						break
					}

					query[key] = append(query[key], value)
				}

				if i < len(values)-1 {
					query[key] = append(query[key], values[i+1:]...)
				}
			}
		} else {
			query[key] = values
		}
	}

	if len(query) > 0 {
		pathResult += "?" + query.Encode()
	}

	finalRoute, finalParams, reloadOn := r.runMaps(req, route, params)
	return pathResult, r.resolve(req, finalRoute, finalParams, reloadOn)
}

func (r *LocalRouter) resolve(req *http.Request, route string, params Params, reloadOn []string) RouteResult {
	splitRoute := strings.Split(route, ".")
	result := RouteResult{}

	if len(reloadOn) > 0 {
		result.ReloadOn = reloadOn
	}

	for {
		if handlers, ok := r.handlers[route]; ok {
			for _, handler := range handlers {
				partialResult := handler(req, route, params)
				result.ReloadOn = append(result.ReloadOn, partialResult.ReloadOn...)
				result.Controllers = append(result.Controllers, partialResult.Controllers...)
			}
		}

		if len(splitRoute) <= 1 {
			break
		}

		splitRoute = splitRoute[:len(splitRoute)-1]
		route = strings.Join(splitRoute, ".")
	}

	return result
}

func (r *LocalRouter) handleRoute(route string, handler RouteHandlerFunc) {
	r.handlers[route] = append(r.handlers[route], handler)
}

func StaticHandler(h string) RouteHandlerFunc {
	return func(r *http.Request, route string, params Params) RouteResult {
		return RouteResult{
			Controllers: []ControllerPlan{
				{
					Controller: h,
				},
			},
		}
	}
}

func ControllerHandler(h RouteController) RouteHandlerFunc {
	allowedParams := map[string]bool{}
	for _, p := range h.Params {
		allowedParams[p] = true
	}

	return func(r *http.Request, route string, params Params) RouteResult {
		filteredParams := Params{}
		for key, value := range params {
			if allowedParams[key] {
				filteredParams[key] = value
			}
		}

		plan := ControllerPlan{
			Controller: h.Controller,
			Method:     h.Method,
			Params:     filteredParams,

			DependsOn:       h.DependsOn,
			RunAfter:        h.RunAfter,
			Batch:           h.Batch,
			Persistent:      h.Persistent,
			Lazy:            h.Lazy,
			Socket:          h.Socket,
			NeedsCleanup:    h.NeedsCleanup,
			NeedsValidation: h.NeedsValidation,
			Cache:           h.Cache,
			Prefetch:        h.Prefetch,
		}

		return RouteResult{
			Controllers: []ControllerPlan{plan},
		}
	}
}

func ControllerPlanHandler(h ControllerPlan) RouteHandlerFunc {
	return func(r *http.Request, route string, params Params) RouteResult {
		return RouteResult{
			Controllers: []ControllerPlan{h},
		}
	}
}

func (r *LocalRouter) HandleRoute(route string, handlers ...RouteHandlerFunc) {
	for _, handler := range handlers {
		r.handleRoute(route, handler)
	}
}

type RouteHandlers = map[string][]RouteHandlerFunc

func (r *LocalRouter) HandleRoutes(handlers RouteHandlers) {
	for route, h := range handlers {
		r.HandleRoute(route, h...)
	}
}
