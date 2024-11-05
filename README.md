# Bolt

A fast and easy-to-use Go router.

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
app.RegisterRouteParamValidator("webp", func(value string) (string, error) {
	if strings.HasSuffix(value, ".webp") {
		return strings.TrimSuffix(value, ".webp"), nil
	}
	return "", errors.New("invalid file")
})
```

Use a parameter validation:
```go
app.Get("/images/{image@webp}", func(c bolt.Ctx) error {
	return c.JSON(bolt.Map{
		"image_name": c.Param("image"), // it will return the image name without the .webp extension.
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

Bolt uses https://github.com/coder/websocket package for handling websocket connections.
We wrapped our `Ctx` and `*websocket.Conn` into a `WSConn` struct for easier usage.

You can create a websocket connection like this:
```go
server := bolt.NewWSServer(context.Background())

// This must be set. Here you can define your own websocket connection handler.
server.OnMessage(func(s bolt.WSServer, conn bolt.WSConn, msg []byte) error {})
```

You don't have to set up loops are anything. The server will handle everything for you.
You only need to handle the incoming message and broadcast to your desired clients.

```go
app.WS("/ws", func(c WSConn) {
	// Handle the connection here with a loop by itself or add it to the server.
	server.AddConn(c)
})
```
⚠️ Attention: The `WS` method is just like a `Get` method with a special adapter used for websocket connections.
You can't specify the same route for both `Get` and `WS` methods.

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

### Hooks with Bolt

Bolt supports hooks. Hooks are functions that are called on a specific event. You can register a hook for a specific event.

```go
app.Hook(bolt.PreRequestHook, func(c Ctx) error {
	// A simple hook handler with an error return.
	return nil // or just return nil to continue the request chain.
})
```

You can also use `bolt.PostRequestHook` that runs after the request handled.
