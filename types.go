package bolt

// HandlerFunc is a function that handles a request.
type HandlerFunc func(c Ctx) error

// MiddlewareFunc is a function that is executed before the handler.
type MiddlewareFunc func(c Ctx) (bool, error)

// WSHandlerFunc is a function that handles a WebSocket request.
type WSHandlerFunc func(conn WSConn)

// RouteParamValidatorFunc is a function that validates a route parameter.
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
	PreRequestHook  BoltHook = iota
	PostRequestHook BoltHook = iota
)
