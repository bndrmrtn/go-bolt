package bolt

import (
	"context"
	"fmt"
	"io"
	"log"
	"net/http"
	"testing"
	"time"
)

func TestServer(t *testing.T) {
	app := New()
	server := NewWSServer(context.Background())

	server.OnMessage(func(s WSServer, conn WSConn, msg []byte) error {
		fmt.Println("New message:", string(msg))
		return conn.SendJSON(Map{
			"echo": string(msg),
		})
	})

	app.Hook(PreRequestHook, func(c Ctx) error {
		return nil
	})

	app.Get("/test", func(c Ctx) error {
		return c.Status(201).JSON(Map{
			"hello": "world",
		})
	})

	// Simply response to an endpoint
	app.Get("/", func(c Ctx) error {
		user, err := c.Session().Get("user")
		if err != nil {
			return err
		}

		return c.JSON(Map{
			"user": string(user),
		})
	}).Name("index")

	app.Get("/set", func(c Ctx) error {
		return c.Session().Set("user", []byte("John Doe"))
	})

	// Get a path parameter with "bool" validator
	app.Get("/status/{status@bool}", func(c Ctx) error {
		return c.JSON(Map{
			"Status": c.Param("status"),
		})
	})

	// Get a user with username or empty parameter
	// This will create multiple routes for the same handler
	// -> /user/{username}
	// -> /user -> here the handler will response with "Unknown" unless defaultValue is not specified
	app.Get("/user/{username}?", func(c Ctx) error {
		return c.JSON(Map{
			"user": c.Param("username", "Unknown"),
		})
	})

	// Sending a response pipe
	app.Get("/pipe", func(c Ctx) error {
		return c.Pipe(func(pw *io.PipeWriter) {
			// Pipe is closed automatically
			for i := 0; i < 5; i++ {
				pw.Write([]byte(fmt.Sprintf("Streaming data chunk %d\n", i)))
				time.Sleep(1 * time.Second) // Simulate some delay
			}
		})
	})

	// Basic middleware usage
	app.Get("/secret", func(c Ctx) error {
		return c.SendString("Secret data")
	}, func(c Ctx) (bool, error) {
		if c.URL().Query().Get("auth") == "123" {
			return true, nil
		}

		return false, c.Status(http.StatusUnauthorized).SendString("Unauthorized")
	})

	group := app.Group("/group")
	// EasyFastAdaptor is a helper function to convert EasyFastHandlerFunc to HandlerFunc
	group.Get("/", EasyFastAdaptor(func(c Ctx) (any, error) {
		return "Hello bello", nil
	}))

	group.Get("/map", EasyFastAdaptor(func(c Ctx) (any, error) {
		return Map{
			"message": "Hello",
		}, nil
	}))

	app.Post("/sayHello", EasyFastAdaptor(func(c Ctx) (any, error) {
		data := struct {
			Username string `json:"username"`
		}{}
		if err := c.Body().ParseJSON(&data); err != nil {
			return nil, err
		}
		return Map{
			"message": fmt.Sprintf("Hello %s!", data.Username),
		}, nil
	}))

	app.WS("/ws", func(conn WSConn) {
		server.AddConn(conn)
	})

	app.Dump()
	log.Fatal(app.Serve(":3001"))
}
