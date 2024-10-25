package bolt

import (
	"errors"
	"html/template"
	"net/http"
	"time"

	"github.com/google/uuid"
)

// Mode is the application mode
type Mode string

const (
	Production  Mode = "production"
	Development Mode = "development"
)

// Config is the configuration of the Bolt application
type Config struct {
	// ErrorHandler handles the request errors
	ErrorHandler func(c Ctx, err error) error
	// NotFoundHandler handles the not found requests
	NotFoundHandler func(c Ctx) error
	// Mode is the application mode
	// default is development
	Mode Mode

	Views     *ViewConfig
	Session   *SessionConfig
	Websocket *WSConfig

	// Auth map[string]MiddlewareFunc // bolt.Auth("session-default")
}

// WSConfig is the configuration of the websocket
type WSConfig struct {
	Timeout time.Duration
}

type ViewConfig struct {
	Views *template.Template
}

// SessionConfig is the configuration of the session
type SessionConfig struct {
	// Enabled is a flag to enable or disable the session
	// session is enabled by default
	Enabled bool
	// TokenFunc is a function to get the session token
	// by default it generates a new token if not exists, and stores it in a cookie named "session"
	TokenFunc func(c Ctx) (string, error)
	// TokenExpire is the session token expiration time
	// by default it is 12 hours
	// tokens are renewed at each modification
	TokenExpire time.Duration
	// Store is the session storage
	// by default it uses the MemStorage, an in-memory storage
	Store SessionStore
}

func (c *Config) check() {
	if c.ErrorHandler == nil {
		c.ErrorHandler = defaultErrorHandler
	}

	if c.NotFoundHandler == nil {
		c.NotFoundHandler = defaultNotFoundHandler
	}

	if c.Mode == "" {
		c.Mode = Development
	}

	if c.Websocket == nil {
		c.Websocket = defaultWSConfig()
	}

	if c.Session == nil {
		c.Session = defaultSessionConfig()
	}
}

func (s *SessionConfig) check() {
	if s.TokenFunc == nil {
		s.TokenFunc = defaultTokenFunc
	}

	if s.Store == nil {
		s.Store = NewMemStorage()
	}

	if s.TokenExpire == 0 {
		s.TokenExpire = time.Hour * 12
	}
}

func defaultConfig() *Config {
	return &Config{
		ErrorHandler:    defaultErrorHandler,
		NotFoundHandler: defaultNotFoundHandler,
		Mode:            Development,
		Websocket:       defaultWSConfig(),
		Session:         defaultSessionConfig(),
	}
}

type Error struct {
	err    string
	status int
}

func NewError(statusCode int, err string) error {
	return &Error{
		err:    err,
		status: statusCode,
	}
}

func (h *Error) Error() string {
	return h.err
}

func defaultErrorHandler(c Ctx, err error) error {
	code := http.StatusInternalServerError

	var e *Error
	if errors.As(err, &e) {
		code = e.status
	}

	c.Status(code)
	return c.Format(Map{"error": err.Error()})
}

func defaultNotFoundHandler(c Ctx) error {
	return NewError(http.StatusNotFound, "Not found")
}

func defaultWSConfig() *WSConfig {
	return &WSConfig{
		Timeout: time.Second * 10,
	}
}

func defaultSessionConfig() *SessionConfig {
	return &SessionConfig{
		Enabled:     true,
		TokenExpire: time.Hour * 12,
		TokenFunc:   defaultTokenFunc,
		Store:       NewMemStorage(),
	}
}

func defaultTokenFunc(c Ctx) (string, error) {
	cookie, err := c.Cookie().Get("session")
	if err != nil {
		token := uuid.New().String()
		c.Cookie().Set(&http.Cookie{
			Name:    "session",
			Value:   token,
			Expires: time.Now().Add(time.Hour * 12),
		})
		return token, nil
	}
	return cookie.Value, nil
}

func defaultViewConfig() *ViewConfig {
	return nil
}
