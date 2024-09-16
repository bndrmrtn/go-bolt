package bolt

import (
	"encoding/json"
	"encoding/xml"
	"errors"
	"fmt"
	"io"
	"mime/multipart"
	"net"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strconv"
	"strings"

	"github.com/fatih/color"
)

// Map is a map[string]any alias to make it more readable
type Map map[string]any

// Ctx is the context of the request
type Ctx interface {
	// Method returns the request method
	Method() string
	// URL returns the request URL
	URL() *url.URL
	// Path returns the request path
	Path() string
	// Params returns all route params
	Params() map[string]string
	// Param returns a route param by name
	Param(name string, defaultValue ...string) string
	// ParamInt returns a route param by name as int
	ParamInt(name string, defaultValue ...int) (int, error)

	// ResponseWriter returns the http.ResponseWriter
	ResponseWriter() http.ResponseWriter
	// Request returns the http.Request
	Request() *http.Request
	// App returns the Bolt application
	App() *Bolt
	// IP returns the client IP
	IP() net.IP

	// Headers
	// Header returns a HeaderCtx to add response header and get request header
	Header() HeaderCtx
	// Cookie returns a CookieCtx to get and set cookies
	Cookie() CookieCtx
	// Session returns a SessionCtx to get and set session data (if session is enabled in the configuration)
	Session() SessionCtx
	// ContentType sets the response content type
	ContentType(t string) Ctx

	// Response

	// Status sets the response status code
	Status(code int) Ctx
	// Send sends the output as a byte slice
	Send([]byte) error
	// SendString sends the output as a string
	SendString(s string) error
	// JSON sends the output as a JSON
	JSON(data any) error
	// XML sends the output as a XML
	XML(data any) error
	// SendFile sends a file as a response
	SendFile(path string) error
	// Pipe sends the output as a stream
	Pipe(pipe func(pw *io.PipeWriter)) error
	// Format sends the output in the format specified in the Accept header
	Format(data any) error
	// Redirect redirects the request to the specified URL
	Redirect(to string) error

	// Request

	// Body returns a BodyCtx to parse the request body
	Body() BodyCtx

	// Utils

	// Get returns a stored value by key
	Get(key string) any
	// Set stores a value by key in the context (useful for middleware)
	// note: the value only exists in the current request
	Set(key string, value any)
}

// HeaderCtx is the context of the request headers
type HeaderCtx interface {
	// Add adds a header to the response
	Add(key, value string)
	// Get returns a header from the request
	Get(key string) string
}

// BodyCtx is the context of the request body
type BodyCtx interface {
	// Parse the request body to any by the Content-Type header
	Parse(v any) error
	// ParseJSON parses the request body as JSON
	ParseJSON(v any) error
	// ParseXML parses the request body as XML
	ParseXML(v any) error
	// ParseForm parses the request body as form
	ParseForm(v any) error
	// File returns a file from the request
	File(name string, maxSize ...int) (multipart.File, *multipart.FileHeader, error)
}

// CookieCtx is the context of the request cookies
type CookieCtx interface {
	// Get returns a cookie by name
	Get(name string) (*http.Cookie, error)
	// Set sets a cookie
	Set(cookie *http.Cookie)
}

// SessionCtx is the context of the request session
type SessionCtx interface {
	// Get returns a session value by key
	Get(key string) ([]byte, error)
	// Set sets a session value by key
	Set(key string, value []byte) error
	// Delete deletes a session value by key
	Delete(key string) error
	// Destroy destroys the session
	Destroy() error
}

// Implementing the Ctx

type ctx struct {
	b           *Bolt
	route       Route
	routeParams map[string]string

	w http.ResponseWriter
	r *http.Request

	statusCode int
	headers    map[string][]string

	store map[string]any
}

func newCtx(b *Bolt, route Route, w http.ResponseWriter, r *http.Request, routeParams map[string]string) Ctx {
	return &ctx{
		b:           b,
		route:       route,
		routeParams: routeParams,
		w:           w,
		r:           r,
		statusCode:  200,
		headers:     make(map[string][]string),
		store:       make(map[string]any),
	}
}

func (c *ctx) App() *Bolt {
	return c.b
}

func (c *ctx) IP() net.IP {
	return net.IP(c.r.RemoteAddr)
}

func (c *ctx) ResponseWriter() http.ResponseWriter {
	return c.w
}

func (c *ctx) Request() *http.Request {
	return c.r
}

func (c *ctx) Method() string {
	return c.r.Method
}

func (c *ctx) URL() *url.URL {
	return c.r.URL
}

func (c *ctx) Path() string {
	return c.r.URL.Path
}

func (c *ctx) Params() map[string]string {
	return c.routeParams
}

func (c *ctx) Param(name string, defaultValue ...string) string {
	param, ok := c.routeParams[name]
	if !ok {
		if len(defaultValue) > 0 {
			return defaultValue[0]
		}
		return ""
	}
	return param
}

func (c *ctx) ParamInt(name string, defaultValue ...int) (int, error) {
	param, ok := c.routeParams[name]
	if !ok {
		if len(defaultValue) > 0 {
			return defaultValue[0], errors.New("param not found")
		}
		return 0, errors.New("param not found")
	}

	return strconv.Atoi(param)
}

func (c *ctx) Header() HeaderCtx {
	return &headerCtx{
		c: c,
	}
}

func (c *ctx) ContentType(t string) Ctx {
	c.Header().Add("Content-Type", t)
	return c
}

func (c *ctx) Status(code int) Ctx {
	c.statusCode = code
	return c
}

func (c *ctx) Send(b []byte) error {
	c.writeHeaders()
	_, err := c.w.Write(b)
	return err
}

func (c *ctx) SendString(s string) error {
	return c.Send([]byte(s))
}

func (c *ctx) JSON(data any) error {
	b, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return c.ContentType(ContentTypeJSON).Send(b)
}

func (c *ctx) XML(data any) error {
	b, err := xml.Marshal(data)
	if err != nil {
		return err
	}

	return c.ContentType(ContentTypeXML).Send(b)
}

func (c *ctx) SendFile(path string) error {
	file, err := os.Open(path)
	if err != nil {
		return err
	}

	c.writeHeaders()
	_, err = io.Copy(c.w, file)
	return err
}

func (c *ctx) Pipe(pipe func(pw *io.PipeWriter)) error {
	pr, pw := io.Pipe()

	go func(pw *io.PipeWriter) {
		defer pw.Close()
		pipe(pw)
	}(pw)

	c.writeHeaders()
	_, err := io.Copy(c.w, pr)
	return err
}

func (c *ctx) Format(data any) error {
	var allowedFormats = []string{ContentTypeJSON, ContentTypeText, ContentTypeHTML, ContentTypeXML}
	format := c.getHeaderAllowedFormat(allowedFormats, ContentTypeJSON)

	var d string
	switch data.(type) {
	case string:
		d = data.(string)
	case []byte:
		d = string(data.([]byte))
	default:
		d = fmt.Sprintf("%v", data)
	}

	switch format {
	case ContentTypeJSON:
		return c.JSON(data)
	case ContentTypeText:
		return c.SendString(d)
	case ContentTypeHTML:
		return c.ContentType(ContentTypeHTML).SendString("<p>" + d + "</p>")
	case ContentTypeXML:
		return c.XML(data)
	}

	return c.SendString(d)
}

func (c *ctx) Redirect(to string) error {
	if !slices.Contains([]int{http.StatusPermanentRedirect, http.StatusTemporaryRedirect}, c.statusCode) {
		c.Status(http.StatusTemporaryRedirect)
	}
	c.Header().Add("Location", to)
	c.writeHeaders()
	return nil
}

func (c *ctx) Body() BodyCtx {
	return &bodyCtx{
		c: c,
	}
}

func (c *ctx) Get(key string) any {
	return c.store[key]
}

func (c *ctx) Set(key string, value any) {
	c.store[key] = value
}

func (c *ctx) Cookie() CookieCtx {
	return &cookieCtx{
		c: c,
	}
}

func (c *ctx) Session() SessionCtx {
	if !c.b.config.Session.Enabled {
		color.Red("🛑 Sessions are disabled, please enable it in the config or do not use the session context")
		os.Exit(1)
	}
	return &sessionCtx{
		c: c,
	}
}

func (c *ctx) writeHeaders() {
	for header, values := range c.headers {
		for _, value := range values {
			c.w.Header().Add(header, value)
		}
	}

	c.w.WriteHeader(c.statusCode)
}

func (c *ctx) getHeaderAllowedFormat(allowed []string, defaultValue string) string {
	acceptHeader := c.r.Header.Get("Accept")
	if acceptHeader == "" {
		return defaultValue
	}

	wants := strings.Split(acceptHeader, ",")
	for _, format := range wants {
		if slices.Contains(allowed, format) {
			return format
		}
	}

	return defaultValue
}

// Implementing the SessionCtx

type sessionCtx struct {
	c *ctx
}

func newSessionCtx(c *ctx) *sessionCtx {
	return &sessionCtx{
		c: c,
	}
}

func (s *sessionCtx) Get(key string) ([]byte, error) {
	conf := s.c.b.config.Session

	token, err := conf.TokenFunc(s.c)
	if err != nil {
		return nil, err
	}

	b, err := conf.Store.Get(token)
	if err != nil {
		return nil, err
	}

	data := make(map[string][]byte)
	if err := json.Unmarshal(b, &data); err != nil {
		return nil, err
	}
	out, ok := data[key]
	if !ok {
		return nil, fmt.Errorf("key %s not found", key)
	}
	return out, nil
}

func (s *sessionCtx) Set(key string, value []byte) error {
	conf := s.c.b.config.Session

	token, err := conf.TokenFunc(s.c)
	if err != nil {
		return err
	}

	data := make(map[string][]byte)

	b, err := conf.Store.Get(token)
	if err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			return err
		}
	}

	data[key] = value

	b, err = json.Marshal(data)
	if err != nil {
		return err
	}

	return conf.Store.SetEx(token, b, conf.TokenExpire)
}

func (s *sessionCtx) Delete(key string) error {
	conf := s.c.b.config.Session

	token, err := conf.TokenFunc(s.c)
	if err != nil {
		return err
	}

	data := make(map[string][]byte)

	b, err := conf.Store.Get(token)
	if err == nil {
		if err := json.Unmarshal(b, &data); err != nil {
			return err
		}
	}

	delete(data, key)

	b, err = json.Marshal(data)
	if err != nil {
		return err
	}

	return conf.Store.SetEx(token, b, conf.TokenExpire)
}

func (s *sessionCtx) Destroy() error {
	conf := s.c.b.config.Session

	token, err := conf.TokenFunc(s.c)
	if err != nil {
		return err
	}

	return conf.Store.Del(token)
}

// Implementing the CookieCtx

type cookieCtx struct {
	c *ctx
}

func newCookieCtx(c *ctx) CookieCtx {
	return &cookieCtx{
		c: c,
	}
}

func (c *cookieCtx) Get(name string) (*http.Cookie, error) {
	return c.c.r.Cookie(name)
}

func (c *cookieCtx) Set(cookie *http.Cookie) {
	http.SetCookie(c.c.w, cookie)
}

// Implementing the HeaderCtx

type headerCtx struct {
	c *ctx
}

func (h *headerCtx) Add(key, value string) {
	h.c.headers[key] = append(h.c.headers[key], value)
}

func (h *headerCtx) Get(key string) string {
	return h.c.r.Header.Get(key)
}

// Implementing the BodyCtx

type bodyCtx struct {
	c *ctx
}

func (b *bodyCtx) ParseJSON(v any) error {
	return json.NewDecoder(b.c.r.Body).Decode(v)
}

func (b *bodyCtx) ParseForm(v any) error {
	return NewError(http.StatusNotImplemented, "Not implemented")
}

func (b *bodyCtx) ParseXML(v any) error {
	return xml.NewDecoder(b.c.r.Body).Decode(v)
}

func (b *bodyCtx) Parse(v any) error {
	switch b.c.r.Header.Get("Content-Type") {
	case ContentTypeJSON:
		return b.ParseJSON(v)
	case ContentTypeXML:
		return b.ParseXML(v)
	case ContentTypeForm:
		return b.ParseForm(v)
	}

	return NewError(http.StatusUnprocessableEntity, "Unprocessable Entity")
}

func (b *bodyCtx) File(name string, maxSize ...int) (multipart.File, *multipart.FileHeader, error) {
	var size = 10
	if len(maxSize) > 0 {
		size = maxSize[0]
	}

	err := b.c.r.ParseMultipartForm(int64(size) << 20)
	if err != nil {
		return nil, nil, err
	}

	return b.c.r.FormFile(name)
}
