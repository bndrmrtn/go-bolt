package bolt

import (
	"errors"
	"net/http"
	"os"
	"path"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

type fullRouter interface {
	Router
	RouterParamValidator
	exportRoutes() []*route
	getValidator(name string) (RouteParamValidatorFunc, error)
}

type Router interface {
	Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc)
	Add(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc)

	WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc)

	Group(prefix string, middlewares ...MiddlewareFunc) Router
}

type RouterParamValidator interface {
	RegisterRouteParamValidator(name string, fn RouteParamValidatorFunc)
}

type route struct {
	name        string
	method      string
	rawPath     string
	handler     HandlerFunc
	middlewares []MiddlewareFunc
}

// Implement the router

type router struct {
	routes     []*route
	validators map[string]RouteParamValidatorFunc
}

func newRouter() fullRouter {
	return &router{
		routes:     []*route{},
		validators: map[string]RouteParamValidatorFunc{},
	}
}

func (r *router) Add(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	for _, p := range createOptionalRoutes(path) {
		if r.routeExists(method, p) {
			color.Red("Error: route \"%s %s\" already exists.", method, p)
			os.Exit(1)
		}

		r.routes = append(r.routes, &route{
			method:      method,
			rawPath:     p,
			handler:     handler,
			middlewares: middlewares,
		})
	}
}

func (r *router) routeExists(method, path string) bool {
	for _, route := range r.routes {
		if route.method == method && route.rawPath == path {
			return true
		}
	}

	return false
}

func (r *router) Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodGet, path, handler, middlewares...)
}

func (r *router) Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPost, path, handler, middlewares...)
}

func (r *router) Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPut, path, handler, middlewares...)
}

func (r *router) Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodDelete, path, handler, middlewares...)
}

func (r *router) Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPatch, path, handler, middlewares...)
}

func (r *router) Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodOptions, path, handler, middlewares...)
}

func (r *router) All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add("*", path, handler, middlewares...)
}

func (r *router) WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodGet, path, wsHandler(handler), middlewares...)
}

func (r *router) Group(prefix string, middlewares ...MiddlewareFunc) Router {
	return &routerGroup{
		router:      r,
		middlewares: middlewares,
		prefix:      prefix,
	}
}

func (r *router) RegisterRouteParamValidator(name string, fn RouteParamValidatorFunc) {
	r.validators[name] = fn
}

func (r *router) getValidator(name string) (RouteParamValidatorFunc, error) {
	v, ok := r.validators[name]
	if !ok {
		return nil, errors.New("validator '" + name + "' does not exists")
	}
	return v, nil
}

func (r *router) exportRoutes() []*route {
	return r.routes
}

type routerGroup struct {
	router      *router
	middlewares []MiddlewareFunc
	prefix      string
}

func (r *routerGroup) Add(method, p string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.router.Add(method, path.Join(r.prefix, p), handler, append(r.middlewares, middlewares...)...)
}

func (r *routerGroup) Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodGet, path, handler, middlewares...)
}

func (r *routerGroup) Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPost, path, handler, middlewares...)
}

func (r *routerGroup) Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPut, path, handler, middlewares...)
}

func (r *routerGroup) Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodDelete, path, handler, middlewares...)
}

func (r *routerGroup) Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodPatch, path, handler, middlewares...)
}

func (r *routerGroup) Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodOptions, path, handler, middlewares...)
}

func (r *routerGroup) All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add("*", path, handler, middlewares...)
}

func (r *routerGroup) WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc) {
	r.Add(http.MethodGet, path, wsHandler(handler), middlewares...)
}

func (r *routerGroup) Group(prefix string, middlewares ...MiddlewareFunc) Router {
	return &routerGroup{
		router:      r.router,
		prefix:      path.Join(r.prefix, prefix),
		middlewares: middlewares,
	}
}

func createOptionalRoutes(route string) []string {
	var routes []string

	re := regexp.MustCompile(`\{[^\}]+\}\?`)

	matches := re.FindAllString(route, -1)

	if len(matches) > 0 {
		routeWithParams := strings.Replace(route, "?", "", -1)
		routes = append(routes, routeWithParams)

		for _, match := range matches {
			routeWithoutParam := strings.Replace(route, match, "", 1)

			routeWithoutParam = strings.Replace(routeWithoutParam, "//", "/", -1)

			if routeWithoutParam != "/" && strings.HasSuffix(routeWithoutParam, "/") {
				routeWithoutParam = strings.TrimSuffix(routeWithoutParam, "/")
			}

			routeWithoutParam = strings.Replace(routeWithoutParam, "?", "", -1)

			routes = append(routes, routeWithoutParam)
		}
	} else {
		routes = append(routes, route)
	}

	return routes
}
