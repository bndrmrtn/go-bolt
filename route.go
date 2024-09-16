package bolt

import (
	"os"
	"regexp"
	"strings"

	"github.com/fatih/color"
)

type Route interface {
	Name(name string)
	GetName() string
	Method() string
	Path() string
	NormalizedPaths() []string

	Handler() HandlerFunc
	Middlewares() []MiddlewareFunc

	// allRoute returns all possible routes and their parts
	allRoutesParts() [][]RoutePart
	comparePath(router CompleteRouter, path string) (bool, map[string]string)
}

type RoutePart struct {
	Static     bool     `json:"static"`
	Value      string   `json:"value"`
	Validators []string `json:"validators,omitempty"`
}

// Implementing route

type route struct {
	name        string
	method      string
	rawPath     string
	paths       []string
	handler     HandlerFunc
	middlewares []MiddlewareFunc
	parts       [][]RoutePart
}

func newRoute(method, path string, handler HandlerFunc, middlewares []MiddlewareFunc) Route {
	routePaths := createOptionalRoutes(path)
	r := &route{
		method:      method,
		rawPath:     path,
		paths:       routePaths,
		handler:     handler,
		middlewares: middlewares,
		parts:       make([][]RoutePart, len(routePaths)),
	}
	r.parse()
	return r
}

func (r *route) Name(name string) {
	r.name = name
}

func (r *route) GetName() string {
	return r.name
}

func (r *route) Method() string {
	return r.method
}

func (r *route) Path() string {
	return r.rawPath
}

func (r *route) NormalizedPaths() []string {
	return []string{r.rawPath}
}

func (r *route) allRoutesParts() [][]RoutePart {
	return r.parts
}

func (r *route) Handler() HandlerFunc {
	return r.handler
}

func (r *route) Middlewares() []MiddlewareFunc {
	return r.middlewares
}

func (r *route) parse() {
	for _, path := range r.paths {
		p := r.parsePath(path)
		r.parts = append(r.parts, p)
	}
}

func (r *route) parsePath(path string) []RoutePart {
	path = strings.TrimSpace(strings.Trim(path, "/"))
	parts := []RoutePart{}

	for _, part := range strings.Split(path, "/") {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			part = part[1 : len(part)-1]
			var validators []string

			if strings.Contains(part, "@") {
				parts := strings.SplitN(part, "@", 2)
				if len(parts) != 2 {
					color.Red("Invalid route path: '%s'", path)
					os.Exit(1)
				}

				part = parts[0]
				validators = strings.Split(parts[1], ",")
			}

			parts = append(parts, RoutePart{
				Static:     false,
				Value:      part,
				Validators: validators,
			})
		} else {
			parts = append(parts, RoutePart{
				Static: true,
				Value:  part,
			})
		}
	}

	return parts
}

func (r *route) comparePath(router CompleteRouter, path string) (bool, map[string]string) {
	path = strings.Trim(path, "/")
	pathParts := strings.Split(path, "/")

	for _, route := range r.allRoutesParts() {
		if len(pathParts) != len(route) {
			continue
		}

		ok, params := r.compareSinglePath(router, route, pathParts)
		if !ok {
			continue
		}

		return ok, params
	}

	return false, nil
}

func (r *route) compareSinglePath(router CompleteRouter, route []RoutePart, pathParts []string) (bool, map[string]string) {
	params := make(map[string]string)

	if len(route) == 0 && len(pathParts) == 0 {
		return true, nil
	}

	for i, part := range route {
		if part.Static {
			if part.Value != pathParts[i] {
				return false, nil
			}
		} else {
			value := pathParts[i]

			for _, v := range part.Validators {
				fn, err := router.getValidator(v)
				if err != nil {
					color.Red("Validator not found: %s", v)
					os.Exit(1)
				}

				value, err = fn(value)
				if err != nil {
					return false, nil
				}
			}

			params[part.Value] = value
		}
	}

	return true, params
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
