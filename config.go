package bolt

import (
	"errors"
	"net/http"
	"time"

	"github.com/google/uuid"
)

type Mode string

const (
	Production  Mode = "production"
	Development Mode = "development"
)

type Config struct {
	ErrorHandler    func(c Ctx, err error) error
	NotFoundHandler func(c Ctx) error
	Mode            Mode

	Session   *SessionConfig
	Websocket *WSConfig

	// Auth map[string]MiddlewareFunc // bolt.Auth("session-default")
}

type WSConfig struct {
	Timeout time.Duration
}

type SessionConfig struct {
	Enabled   bool
	TokenFunc func(c Ctx) (string, error)
	Store     SessionStore
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
		Enabled: true,
		TokenFunc: func(c Ctx) (string, error) {
			cookie, err := c.Cookie().Get("session")
			if err != nil {
				token := uuid.New().String()
				c.Cookie().Set(&http.Cookie{
					Name:  "session",
					Value: token,
				})
				return token, nil
			}
			return cookie.Value, nil
		},
		Store: NewMemStorage(),
	}
}
