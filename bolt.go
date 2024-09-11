package bolt

import (
	"errors"
	"net/http"
	"os"
	"time"
)

const Version = "0.1.0"

type Bolt struct {
	config *Config

	fullRouter
	start     time.Time
	server    *server
	publicDir string
	hooks     map[BoltHook][]func(c Ctx)
}

func New(conf ...*Config) *Bolt {
	var c *Config
	if len(conf) > 0 {
		c = conf[0]
	} else {
		c = defaultConfig()
	}

	c.check()

	b := &Bolt{
		config:     c,
		fullRouter: newRouter(),
		publicDir:  "./public",
		hooks:      make(map[BoltHook][]func(c Ctx)),
	}
	b.server = &server{b}
	return b
}

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

func (b *Bolt) Hook(hook BoltHook, fn func(c Ctx)) {
	b.hooks[hook] = append(b.hooks[hook], fn)
}

func (b *Bolt) Serve(listenAddr string) error {
	displayServeInfo(listenAddr, b.config.Mode)
	b.start = time.Now()
	return http.ListenAndServe(listenAddr, b.server)
}

func (b *Bolt) ServeTLS(listenAddr, certFile, keyFile string) error {
	displayServeInfo(listenAddr, b.config.Mode)
	b.start = time.Now()
	return http.ListenAndServeTLS(listenAddr, certFile, keyFile, b.server)
}
