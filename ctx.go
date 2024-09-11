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

type Map map[string]any

type Ctx interface {
	Method() string
	URL() *url.URL
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
	// Bolt returns the Bolt application
	Bolt() *Bolt
	IP() net.IP

	// Headers
	// Header returns a HeaderCtx to add response header and get request header
	Header() HeaderCtx
	Cookie() CookieCtx
	Session() SessionCtx
	// ContentType sets the response content type
	ContentType(t string) Ctx

	// Response

	Status(code int) Ctx
	Send([]byte) error
	SendString(s string) error
	JSON(data any) error
	XML(data any) error
	SendFile(path string) error
	// Pipe sends the output as a stream
	Pipe(pipe func(pw *io.PipeWriter)) error
	// Format sends the output in the format specified in the Accept header
	Format(data any) error

	// Request

	Body() BodyCtx

	// Utils

	Get(key string) any
	Set(key string, value any)
}

// Implementing the Ctx

type ctx struct {
	b           *Bolt
	route       *route
	routeParams map[string]string

	w http.ResponseWriter
	r *http.Request

	store map[string]any
}

func newCtx(b *Bolt, route *route, w http.ResponseWriter, r *http.Request, routeParams map[string]string) Ctx {
	return &ctx{
		b:           b,
		route:       route,
		routeParams: routeParams,
		w:           w,
		r:           r,
		store:       make(map[string]any),
	}
}

func (c *ctx) Bolt() *Bolt {
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
	c.w.WriteHeader(code)
	return c
}

func (c *ctx) Send(b []byte) error {
	_, err := c.w.Write(b)
	return err
}

func (c *ctx) SendString(s string) error {
	_, err := c.w.Write([]byte(s))
	return err
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

	_, err = io.Copy(c.w, file)
	return err
}

func (c *ctx) Pipe(pipe func(pw *io.PipeWriter)) error {
	pr, pw := io.Pipe()

	go func(pw *io.PipeWriter) {
		defer pw.Close()
		pipe(pw)
	}(pw)

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

// Other Ctx

type HeaderCtx interface {
	Add(key, value string)
	Get(key string) string
}

type BodyCtx interface {
	// Parse the request body to any by the Content-Type header
	Parse(v any) error
	ParseJSON(v any) error
	ParseXML(v any) error
	ParseForm(v any) error
	File(name string, maxSize ...int) (multipart.File, *multipart.FileHeader, error)
}

type CookieCtx interface {
	Get(name string) (*http.Cookie, error)
	Set(cookie *http.Cookie)
}

type SessionCtx interface {
	Get(key string) ([]byte, error)
	Set(key string, value []byte) error
}

type sessionCtx struct {
	c *ctx
}

type cookieCtx struct {
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

	return conf.Store.Set(token, b)
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

type headerCtx struct {
	c *ctx
}

func (h *headerCtx) Add(key, value string) {
	h.c.w.Header().Add(key, value)
}

func (h *headerCtx) Get(key string) string {
	return h.c.r.Header.Get(key)
}

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
