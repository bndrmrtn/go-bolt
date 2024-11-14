package gale

import (
	"net/http"
	"os"
	"path/filepath"
	"time"
)

type server struct {
	app *Gale
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

			ctx := newCtx(s.app, route, w, r, params)

			for _, hook := range s.app.hooks[EveryRequestHook] {
				err := hook(ctx)
				if err != nil {
					handleError(ctx, s.app.config.ErrorHandler(ctx, err))
				}

				if !ctx.canContinue() {
					return
				}
			}

			s.handleRoute(route, ctx)
			return
		}
	}

	ctx := newCtx(s.app, nil, w, r, nil)

	for _, hook := range s.app.hooks[EveryRequestHook] {
		err := hook(ctx)
		if err != nil {
			handleError(ctx, s.app.config.ErrorHandler(ctx, err))
		}

		if !ctx.canContinue() {
			return
		}
	}

	err := s.app.config.NotFoundHandler(ctx)
	if err != nil {
		handleError(ctx, s.app.config.ErrorHandler(ctx, err))
	}
}

func (s *server) handleRoute(route Route, ctx Ctx) {
	for _, hook := range s.app.hooks[PreRequestHook] {
		err := hook(ctx)
		if err != nil {
			handleError(ctx, s.app.config.ErrorHandler(ctx, err))
			return
		}

		if !ctx.canContinue() {
			return
		}
	}

	defer func(s *server, ctx Ctx) {
		for _, hook := range s.app.hooks[PostRequestHook] {
			_ = hook(ctx)
			if !ctx.canContinue() {
				break
			}
		}
	}(s, ctx)

	for _, m := range route.Middlewares() {
		err := m(ctx)
		if err != nil {
			handleError(ctx, s.app.config.ErrorHandler(ctx, err))
			return
		}

		if !ctx.canContinue() {
			return
		}
	}

	if !ctx.canContinue() {
		return
	}

	err := route.Handler()(ctx)
	if err != nil {
		handleError(ctx, s.app.config.ErrorHandler(ctx, err))
	}
}

func handleError(ctx Ctx, err error) {
	if err != nil {
		http.Error(ctx.ResponseWriter(), err.Error(), 500)
	}
}
