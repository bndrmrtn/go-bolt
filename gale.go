package gale

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
)

// Version is the current version of Gale
const Version = "0.1.2-alpha"

type Gale struct {
	config *Config

	CompleteRouter
	start     time.Time
	server    *server
	publicDir string
	hooks     map[GaleHook][]func(c Ctx) error
}

// New creates a new Gale application with the given configuration
func New(conf ...*Config) *Gale {
	var c *Config
	if len(conf) > 0 {
		c = conf[0]
	} else {
		c = defaultConfig()
	}

	c.check()

	g := &Gale{
		config:         c,
		CompleteRouter: newRouter(),
		publicDir:      "",
		hooks:          make(map[GaleHook][]func(c Ctx) error),
	}
	g.server = &server{g}

	registerDefaultRouteValidators(g)
	return g
}

// PublicDir sets the public directory for the Gale application
func (g *Gale) PublicDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("path is not a directory")
	}

	g.publicDir = path
	return nil
}

// Hook registers a hook for the given hook type
// Note: Hooks are methods that are executed before or after a request is processed
func (g *Gale) Hook(hook GaleHook, fns ...func(c Ctx) error) {
	if len(fns) == 0 {
		color.Red("No functions provided for hook")
		os.Exit(1)
	}
	g.hooks[hook] = append(g.hooks[hook], fns...)
}

// Use registers an extension for the Gale application
func (g *Gale) Use(fn UseExtension) {
	fn.Register(g)
}

// Config returns the configuration of the Gale application
func (g *Gale) Config() *Config {
	return g.config
}

// Router returns the router of the Gale application
func (g *Gale) Router() CompleteRouter {
	return g.CompleteRouter
}

// Serve starts the Gale server on the given address
func (g *Gale) Serve(listenAddr string) error {
	displayServeInfo(listenAddr, g.config.Mode)
	g.start = time.Now()
	return http.ListenAndServe(listenAddr, g.server)
}

// ServeTLS starts the Gale server on the given address with TLS
func (g *Gale) ServeTLS(listenAddr, certFile, keyFile string) error {
	displayServeInfo(listenAddr, g.config.Mode)
	g.start = time.Now()
	return http.ListenAndServeTLS(listenAddr, certFile, keyFile, g.server)
}
