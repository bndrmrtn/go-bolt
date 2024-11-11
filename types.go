package gale

// HandlerFunc is a function that handles a request.
type HandlerFunc func(c Ctx) error

// MiddlewareFunc is a function that is executed before the handler.
type MiddlewareFunc func(c Ctx) error

// WSHandlerFunc is a function that handles a WebSocket request.
type WSHandlerFunc func(conn WSConn)

// RouteParamValidatorFunc is a function that validates a route parameter.
type RouteParamValidatorFunc func(value string) (string, error)

type UseExtension interface {
	Register(g *Gale)
}

const (
	ContentTypeJSON      = "application/json"
	ContentTypeText      = "text/plain"
	ContentTypeHTML      = "text/html"
	ContentTypeXML       = "application/xml"
	ContentTypeForm      = "application/x-www-form-urlencoded"
	ContentTypeMultipart = "multipart/form-data"
)

type GaleHook int

const (
	// PreRequestHook is executed when the router found a match.
	PreRequestHook GaleHook = iota
	// PostRequestHook is executed after the route handler.
	PostRequestHook GaleHook = iota
	// EveryRequestHook is executed on every request.
	EveryRequestHook GaleHook = iota
)
