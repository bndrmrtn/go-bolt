package gale

import (
	"embed"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"sync"
	"time"
)

type UI struct {
	app    *Gale
	store  SessionStore
	prefix string
	mu     sync.Mutex
}

type UIRouteLog struct {
	Method string
	Path   string
	Name   string
	IP     string
	Time   time.Time
}

//go:embed ui/templates/*.html
//go:embed ui/templates/**/*.html
//go:embed ui/css/build.css
var uiFiles embed.FS

func NewUIDevtools() *UI {
	return &UI{
		prefix: "/__dev",
		store:  NewMemStorage(),
	}
}

func (u *UI) Register(g *Gale) {
	u.app = g
	u.registerRoutes()

	g.Hook(PostRequestHook, u.handleLogPaths)
}

func (u *UI) registerRoutes() {
	r := u.app.Router().Group(u.prefix)

	r.Get("/", func(c Ctx) error {
		return u.render(c, "main", nil)
	})

	r.Get("/routes", func(c Ctx) error {
		var normalized []struct {
			Route
			NormalPath string
			ID         string
		}

		for _, routes := range c.App().Router().Export() {
			for _, route := range routes.NormalizedPaths() {
				normalized = append(normalized, struct {
					Route
					NormalPath string
					ID         string
				}{
					Route:      routes,
					NormalPath: route,
					ID:         base64.StdEncoding.EncodeToString([]byte(routes.GetName() + route + routes.Method())),
				})
			}
		}

		return u.render(c, "routes", Map{
			"Routes": normalized,
		})
	})

	r.Get("/config", func(c Ctx) error {
		return u.render(c, "config", Map{
			"Config": c.App().Config(),
		})
	})

	r.Get("/logs", func(c Ctx) error {
		var data []UIRouteLog

		if u.store.Exists("uiLog") {
			raw, err := u.store.Get("uiLog")
			if err != nil {
				return err
			}

			err = json.Unmarshal(raw, &data)
			if err != nil {
				return err
			}
		}

		return u.render(c, "logs", Map{
			"Logs": data,
		})
	})

	r.Get("/style.css", u.handleGetStyle)
}

func (u *UI) handleLogPaths(c Ctx) error {
	u.mu.Lock()
	defer u.mu.Unlock()

	route := c.Route()
	if route == nil {
		return nil
	}

	var data []UIRouteLog

	if u.store.Exists("uiLog") {
		raw, err := u.store.Get("uiLog")
		if err != nil {
			return err
		}

		err = json.Unmarshal(raw, &data)
		if err != nil {
			return err
		}
	}

	data = append([]UIRouteLog{
		{
			Name:   route.GetName(),
			Method: route.Method(),
			Path:   c.Path(),
			IP:     c.IP(),
			Time:   time.Now(),
		},
	}, data...)

	if data, err := json.Marshal(data); err == nil {
		return u.store.Set("uiLog", data)
	} else {
		return err
	}
}

func (u *UI) handleGetStyle(c Ctx) error {
	file, err := uiFiles.Open("ui/css/build.css")
	if err != nil {
		return err
	}

	byte, err := io.ReadAll(file)
	if err != nil {
		return err
	}

	return c.ContentType("text/css").Send(byte)
}

func (u *UI) render(c Ctx, page string, props Map) error {
	if props == nil {
		props = make(Map)
	}

	if props["Title"] == nil {
		props["Title"] = "Gale UI"
	}

	props["StylePath"] = filepath.Join(u.prefix, "style.css")
	props["Prefix"] = u.prefix

	tmpl, err := template.New("layout").ParseFS(uiFiles, "ui/templates/layout.html")
	if err != nil {
		return err
	}

	tmpl, err = tmpl.New("content").Funcs(u.getFuncMap()).ParseFS(uiFiles, fmt.Sprintf("ui/templates/pages/%s.html", page))
	if err != nil {
		return err
	}

	return tmpl.ExecuteTemplate(c.ResponseWriter(), "layout", props)
}

func (u *UI) getFuncMap() template.FuncMap {
	return template.FuncMap{
		"toLower": strings.ToLower,
		"getFuncName": func(fn any) string {
			return runtime.FuncForPC(reflect.ValueOf(fn).Pointer()).Name()
		},
		"prettyJSON": func(v any) template.HTML {
			b, _ := json.MarshalIndent(v, "", "  ")
			if b == nil {
				return template.HTML("<span class=\"nil\">nil</span>")
			}

			return template.HTML(string(b))
		},
	}
}
