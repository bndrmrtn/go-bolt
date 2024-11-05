package bolt

import (
	"errors"
	"net/http"
	"os"
	"path"

	"github.com/fatih/color"
)

// CompleteRouter is an interface that combines the Router and RouterParamValidator interfaces.
type CompleteRouter interface {
	Router
	RouterParamValidator
	// Dump writes the routes to the console.
	Dump()

	exportRoutes() []Route
	getValidator(name string) (RouteParamValidatorFunc, error)
}

// Router is an interface that defines the methods for registering routes.
type Router interface {
	// Get registers a new GET route for a path with matching handler and optional middlewares.
	Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Post registers a new POST route for a path with matching handler and optional middlewares.
	Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Put registers a new PUT route for a path with matching handler and optional middlewares.
	Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Delete registers a new DELETE route for a path with matching handler and optional middlewares.
	Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Patch registers a new PATCH route for a path with matching handler and optional middlewares.
	Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Options registers a new OPTIONS route for a path with matching handler and optional middlewares.
	Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// All registers a new route for a path with matching handler and optional middlewares.
	All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route
	// Add registers a new route for a path with matching handler and optional middlewares.
	Add(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route

	// WS registers a new WebSocket route for a path with matching handler and optional middlewares.
	// Note: WebSocket routes should have a different path then GET routes.
	WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc) Route

	// Group creates a new router group with a common prefix and optional middlewares.
	Group(prefix string, middlewares ...MiddlewareFunc) Router
}

// RouterParamValidator is an interface that allows you to register custom route parameter validators.
type RouterParamValidator interface {
	// RouterParamValidator is an interface that allows you to register custom route parameter validators.
	// default validators: int, bool, uuid, alpha, alphanumeric.
	RegisterRouteParamValidator(name string, fn RouteParamValidatorFunc)
}

// Implement the router

type router struct {
	routes     []Route
	validators map[string]RouteParamValidatorFunc
}

func newRouter() CompleteRouter {
	return &router{
		routes:     []Route{},
		validators: map[string]RouteParamValidatorFunc{},
	}
}

func (r *router) Add(method, path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	if r.routeExists(method, path) {
		color.Red("Error: route \"%s %s\" already exists.", method, path)
		os.Exit(1)
	}

	route := newRoute(method, path, handler, middlewares)
	r.routes = append(r.routes, route)

	return route
}

func (r *router) routeExists(method, path string) bool {
	for _, route := range r.routes {
		if route.Method() == method && route.Path() == path {
			return true
		}
	}

	return false
}

func (r *router) Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodGet, path, handler, middlewares...)
}

func (r *router) Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPost, path, handler, middlewares...)
}

func (r *router) Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPut, path, handler, middlewares...)
}

func (r *router) Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodDelete, path, handler, middlewares...)
}

func (r *router) Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPatch, path, handler, middlewares...)
}

func (r *router) Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodOptions, path, handler, middlewares...)
}

func (r *router) All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add("*", path, handler, middlewares...)
}

func (r *router) WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodGet, path, wsHandler(handler), middlewares...)
}

func (r *router) Group(prefix string, middlewares ...MiddlewareFunc) Router {
	return &routerGroup{
		router:      r,
		middlewares: middlewares,
		prefix:      prefix,
	}
}

func (r *router) Dump() {
	logRoutes(r.routes)
}

func (r *router) RegisterRouteParamValidator(name string, fn RouteParamValidatorFunc) {
	if _, ok := r.validators[name]; ok {
		color.Red("Error: route param validator \"%s\" already exists.", name)
		os.Exit(1)
	}

	r.validators[name] = fn
}

func (r *router) getValidator(name string) (RouteParamValidatorFunc, error) {
	v, ok := r.validators[name]
	if !ok {
		return nil, errors.New("validator '" + name + "' does not exists")
	}
	return v, nil
}

func (r *router) exportRoutes() []Route {
	return r.routes
}

type routerGroup struct {
	router      *router
	middlewares []MiddlewareFunc
	prefix      string
}

func (r *routerGroup) Add(method, p string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.router.Add(method, path.Join(r.prefix, p), handler, append(r.middlewares, middlewares...)...)
}

func (r *routerGroup) Get(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodGet, path, handler, middlewares...)
}

func (r *routerGroup) Post(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPost, path, handler, middlewares...)
}

func (r *routerGroup) Put(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPut, path, handler, middlewares...)
}

func (r *routerGroup) Delete(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodDelete, path, handler, middlewares...)
}

func (r *routerGroup) Patch(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodPatch, path, handler, middlewares...)
}

func (r *routerGroup) Options(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodOptions, path, handler, middlewares...)
}

func (r *routerGroup) All(path string, handler HandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add("*", path, handler, middlewares...)
}

func (r *routerGroup) WS(path string, handler WSHandlerFunc, middlewares ...MiddlewareFunc) Route {
	return r.Add(http.MethodGet, path, wsHandler(handler), middlewares...)
}

func (r *routerGroup) Group(prefix string, middlewares ...MiddlewareFunc) Router {
	return &routerGroup{
		router:      r.router,
		prefix:      path.Join(r.prefix, prefix),
		middlewares: append(r.middlewares, middlewares...),
	}
}
