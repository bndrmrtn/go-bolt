package bolt

import "github.com/coder/websocket"

type HandlerFunc func(c Ctx) error

type MiddlewareFunc func(c Ctx) (bool, error)

type WSHandlerFunc func(c *websocket.Conn)

type RouteParamValidatorFunc func(value string) (string, error)

const (
	ContentTypeJSON      = "application/json"
	ContentTypeText      = "text/plain"
	ContentTypeHTML      = "text/html"
	ContentTypeXML       = "application/xml"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)

type BoltHook int

const (
	PreRequestHook BoltHook = iota
)
