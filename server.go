package bolt

import (
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"
)

type server struct {
	app *Bolt
}

func (s *server) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	start := time.Now()

	// log the current request information in development mode
	defer func(start time.Time, method string, path string, dev bool) {
		if !dev {
			return
		}
		serverLogger(start, method, path)
	}(start, r.Method, r.URL.Path, s.app.config.Mode == Development)

	if s.app.publicDir != "" {
		if stat, err := os.Stat(filepath.Join(s.app.publicDir, r.URL.Path)); err == nil && !stat.IsDir() {
			http.ServeFile(w, r, filepath.Join(s.app.publicDir, r.URL.Path))
			return
		}
	}

	for _, route := range s.app.exportRoutes() {
		if route.method == r.Method || route.method == "*" {
			ok, params := s.comparePath(r.URL.Path, route)
			if !ok {
				continue
			}
			s.handleRoute(w, r, route, params)
			return
		}
	}

	err := s.app.config.NotFoundHandler(newCtx(s.app, nil, w, r, nil))
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func (s *server) comparePath(path string, r *route) (bool, map[string]string) {
	path = strings.Trim(path, "/")
	routePath := strings.Trim(r.rawPath, "/")

	if path == routePath {
		return true, nil
	}

	routeParts := strings.Split(routePath, "/")
	pathParts := strings.Split(path, "/")

	if len(routeParts) != len(pathParts) {
		return false, nil
	}

	params := make(map[string]string)

	for i, part := range routeParts {
		if strings.HasPrefix(part, "{") && strings.HasSuffix(part, "}") {
			key := part[1 : len(part)-1]
			value := pathParts[i]

			if strings.Contains(key, "@") {
				parts := strings.SplitN(key, "@", 2)
				if len(parts) != 2 {
					log.Fatalf("Invalid number of parameters in route: '%s' expected=2 got=%d", r.rawPath, len(parts))
				}

				validator, err := s.app.getValidator(parts[1])
				if err != nil {
					log.Fatal(err)
				}

				key = parts[0]
				value, err = validator(value)
				if err != nil {
					return false, nil
				}
			}

			params[key] = value
			continue
		}

		if part != pathParts[i] {
			return false, nil
		}
	}

	return true, params
}

func (s *server) handleRoute(w http.ResponseWriter, r *http.Request, route *route, params map[string]string) {
	ctx := newCtx(s.app, route, w, r, params)

	for _, hook := range s.app.hooks[PreRequestHook] {
		hook(ctx)
	}

	defer func(s *server, ctx Ctx) {
		for _, hook := range s.app.hooks[PostRequestHook] {
			hook(ctx)
		}
	}(s, ctx)

	for _, m := range route.middlewares {
		ok, err := m(ctx)
		if err != nil {
			s.app.config.ErrorHandler(ctx, err)
			return
		}

		if !ok {
			return
		}
	}

	err := route.handler(ctx)
	if err != nil {
		s.app.config.ErrorHandler(ctx, err)
	}
}
