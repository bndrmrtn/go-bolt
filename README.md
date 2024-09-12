# Bolt

A lightweight, fast and simple Go framework 🔋
Currently under development. Version: 0.1.0

## Installation

Create a new go project and install the package with the following command:
```
go get github.com/bndrmrtn/go-bolt
```

## Usage

### Creating and configuring a new Bolt application

```go
app := bolt.New() // with default configuration
app := bolt.New(&bolt.Config{...}) // with custom configuration
```

### Routing with Bolt

A simple hello world response:
```go
app.Get("/", func(c bolt.Ctx) error {
	return c.Status(http.StatusOK).SendString("Hello, World!")
})
```

A route with a custom parameter:
```go
app.Get("/user/{name}", func(c bolt.Ctx) error {
	return c.SendString("Hello, " + c.Param("name") + "!")
})
```

A route with an optional parameter:
```go
app.Get("/user/{name}?", func(c bolt.Ctx) error {
	// The c.Param() second option can be a default value if the parameter is not provided
	return c.SendString("Hello, " + c.Param("name", "World") + "!")
})
```

Route parameter validation:
```go
app.RegisterRouteParamValidator("bool", func(value string) (string, error) {
	if value == "true" || value == "false" {
		return value, nil
	}
	return "", errors.New("invalid boolean")
})
```

Use a parameter validation:
```go
app.Get("/user/{is_active@bool}", func(c bolt.Ctx) error {
	return c.JSON(bolt.Map{
		"is_active": c.Param("is_active"), // is_active can be "true" or "false" or the route will be marked as not found.
	})
})
```

Piping a response:
```go
app.Get("/pipe", func(c bolt.Ctx) error {
	return c.Pipe(func(pw *io.PipeWriter) {
		for i := 0; i < 5; i++ {
			pw.Write([]byte(fmt.Sprintf("Streaming data chunk %d\n", i)))
			time.Sleep(1 * time.Second) // Simulate some delay
		}
	})
})
```

### Middleware with Bolt

All middleware function comes after the main handler with Bolt:

```go
app.Get(path string, handler bolt.HandlerFunc, middlewares ...bolt.MiddlewareFunc)
```

A simple middleware:
```go
app.Get("/secret", func(c bolt.Ctx) error {
		return c.SendString("Secret data")
	}, func(c bolt.Ctx) (bool, error) {
	if c.URL().Query().Get("auth") == "123" {
		return true, nil
	}
	return false, c.Status(http.StatusUnauthorized).SendString("Unauthorized")
})
```

### Websockets with Bolt

Bolt does not have it's own websocket support.
We used https://github.com/coder/websocket to make it as good as possible.

```go
app.WS("/ws", func(c *websocket.Conn) {
	defer c.CloseNow()
	// ... Read more in the websocket package documentation
})
```
⚠️ Attention: The WS method is just like a Get method, but it is used for websocket connections.
You have to specify a different route for websocket connections.

### Sessions with Bolt

Bolt supports sessions. You can edit the default configuration or use it as it is.
By default, Bolt uses `uuidv4` and cookies to manage `Session ID`-s.
Sessions are stored in an in-memory store. You can change the store by implementing the `bolt.SessionStore` interface.

Setting a session:
```go
app.Get("/session-set", func(c bolt.Ctx) error {
	session := c.Session()
	session.Set("key", []byte("value"))
	return c.SendString("Session created")
})
```

Getting a session:
```go
app.Get("/session-get", func(c bolt.Ctx) error {
	session := c.Session()
	value, _ := session.Get("key")
	return c.Send(value)
})
```

It also supports `session.Delete("key")` and `session.Destroy()` methods.
