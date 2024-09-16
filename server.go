package bolt

import (
	"net/http"
	"os"
	"path/filepath"
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
		method := route.Method()
		if method == r.Method || method == "*" {
			ok, params := route.comparePath(s.app.CompleteRouter, r.URL.Path)
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

func (s *server) handleRoute(w http.ResponseWriter, r *http.Request, route Route, params map[string]string) {
	ctx := newCtx(s.app, route, w, r, params)

	for _, hook := range s.app.hooks[PreRequestHook] {
		hook(ctx)
	}

	defer func(s *server, ctx Ctx) {
		for _, hook := range s.app.hooks[PostRequestHook] {
			hook(ctx)
		}
	}(s, ctx)

	for _, m := range route.Middlewares() {
		ok, err := m(ctx)
		if err != nil {
			s.app.config.ErrorHandler(ctx, err)
			return
		}

		if !ok {
			return
		}
	}

	err := route.Handler()(ctx)
	if err != nil {
		s.app.config.ErrorHandler(ctx, err)
	}
}
