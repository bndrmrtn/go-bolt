package bolt

import (
	"net/http"
)

// Adaptor converts the standard http.HandlerFunc to a bolt.HandlerFunc
func Adaptor(fn http.HandlerFunc) HandlerFunc {
	return func(c Ctx) error {
		w, r := c.ResponseWriter(), c.Request()
		fn(w, r)
		return nil
	}
}

type EasyFastHandlerFunc func(c Ctx) (any, error)

// EasyFastAdaptor is a fast way to just dump data or error as a response without using the Ctx send methods
func EasyFastAdaptor(fn EasyFastHandlerFunc) HandlerFunc {
	return func(c Ctx) error {
		data, err := fn(c)
		if err != nil {
			return err
		}
		return c.Format(data)
	}
}

// SimpleMiddleware is a fast way to create a middleware that just returns a boolean
func SimpleMiddleware(fn func(c Ctx) bool) MiddlewareFunc {
	return func(c Ctx) (bool, error) {
		return fn(c), nil
	}
}
