package bolt

import (
	"errors"
	"net/http"
	"os"
	"time"

	"github.com/fatih/color"
)

// Version is the current version of Bolt
const Version = "0.1.1"

type Bolt struct {
	config *Config

	CompleteRouter
	start     time.Time
	server    *server
	publicDir string
	hooks     map[BoltHook][]func(c Ctx)
}

// New creates a new Bolt application with the given configuration
func New(conf ...*Config) *Bolt {
	var c *Config
	if len(conf) > 0 {
		c = conf[0]
	} else {
		c = defaultConfig()
	}

	c.check()

	b := &Bolt{
		config:         c,
		CompleteRouter: newRouter(),
		publicDir:      "",
		hooks:          make(map[BoltHook][]func(c Ctx)),
	}
	b.server = &server{b}

	registerDefaultRouteValidators(b)
	return b
}

// PublicDir sets the public directory for the Bolt application
func (b *Bolt) PublicDir(path string) error {
	info, err := os.Stat(path)
	if err != nil {
		return err
	}

	if !info.IsDir() {
		return errors.New("path is not a directory")
	}

	b.publicDir = path
	return nil
}

// Hook registers a hook for the given hook type
// Note: Hooks are methods that are executed before or after a request is processed
func (b *Bolt) Hook(hook BoltHook, fns ...func(c Ctx)) {
	if len(fns) == 0 {
		color.Red("No functions provided for hook")
		os.Exit(1)
	}
	b.hooks[hook] = append(b.hooks[hook], fns...)
}

// Serve starts the Bolt server on the given address
func (b *Bolt) Serve(listenAddr string) error {
	displayServeInfo(listenAddr, b.config.Mode)
	b.start = time.Now()
	return http.ListenAndServe(listenAddr, b.server)
}

// ServeTLS starts the Bolt server on the given address with TLS
func (b *Bolt) ServeTLS(listenAddr, certFile, keyFile string) error {
	displayServeInfo(listenAddr, b.config.Mode)
	b.start = time.Now()
	return http.ListenAndServeTLS(listenAddr, certFile, keyFile, b.server)
}
