package gale

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/fatih/color"
)

// Interfaces

type WSMessageHandler func(s WSServer, conn WSConn, msg []byte) error

// WSServer is the interface that wraps the basic methods for a websocket server.
type WSServer interface {
	// AddConnection adds a new connection to the server.
	AddConn(conn WSConn)
	// RemoveConnection removes a connection from the server.
	RemoveConn(conn WSConn)
	// RemoveConnection removes a connection from the server.
	Broadcast(msg []byte) int
	// BroadcastFunc sends a message to all connections that satisfy the condition.
	BroadcastFunc(msg []byte, fn func(conn WSConn) bool) int
	// BroadcastTo sends a message to a specific connection.
	BroadcastTo(msg []byte, conns ...WSConn) int

	// OnMessage sets the function to be called when a message is received.
	OnMessage(fn WSMessageHandler)

	// Close closes the server.
	Close() error

	handleMessage(conn WSConn, msg []byte)
}

type WSConn interface {
	Ctx() Ctx
	Conn() *websocket.Conn
	Close() error

	Send(data []byte) error
	SendJSON(v any) error

	readLoop(s WSServer)
}

type wsServer struct {
	conns          []WSConn
	messageHandler WSMessageHandler
}

func NewWSServer() WSServer {
	s := &wsServer{
		conns: []WSConn{},
	}

	return s
}

func (s *wsServer) OnMessage(fn WSMessageHandler) {
	s.messageHandler = fn
}

func (s *wsServer) handleMessage(conn WSConn, msg []byte) {
	if s.messageHandler == nil {
		color.Red("No message handler set")
		os.Exit(1)
	}
	err := s.messageHandler(s, conn, msg)
	if err != nil {
		fmt.Println(err)
	}
}

func (s *wsServer) AddConn(conn WSConn) {
	s.conns = append(s.conns, conn)
	go conn.readLoop(s)
}

func (s *wsServer) RemoveConn(conn WSConn) {
	for i, c := range s.conns {
		if c == conn {
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			break
		}
	}
}

func (s *wsServer) Broadcast(msg []byte) int {
	var failed int
	for _, conn := range s.conns {
		err := conn.Conn().Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			failed++
		}
	}
	return len(s.conns) - failed
}

func (s *wsServer) BroadcastFunc(msg []byte, fn func(conn WSConn) bool) int {
	var (
		succeed int
		failed  int
	)

	for _, conn := range s.conns {
		if fn(conn) {
			succeed++
			err := conn.Conn().Write(context.Background(), websocket.MessageText, msg)
			if err != nil {
				failed++
			}
		}
	}

	return succeed - failed
}

func (s *wsServer) BroadcastTo(msg []byte, conns ...WSConn) int {
	var failed int

	for _, conn := range conns {
		err := conn.Conn().Write(context.Background(), websocket.MessageText, msg)
		if err != nil {
			failed++
		}
	}

	return len(conns) - failed
}

func (s *wsServer) Close() error {
	for _, conn := range s.conns {
		conn.Close()
	}
	return nil
}

// Implementing WSConn

type wsConn struct {
	ctx    Ctx
	conn   *websocket.Conn
	quitch chan struct{}
	mu     *sync.Mutex
}

func newWSConn(ctx Ctx, conn *websocket.Conn) WSConn {
	return &wsConn{
		conn:   conn,
		quitch: make(chan struct{}),
		mu:     &sync.Mutex{},
	}
}

func (c *wsConn) Ctx() Ctx {
	return c.ctx
}

func (c *wsConn) Conn() *websocket.Conn {
	return c.conn
}

func (c *wsConn) Send(data []byte) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	return c.conn.Write(context.Background(), websocket.MessageText, data)
}

func (c *wsConn) SendJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}
	return c.Send(b)
}

func (c *wsConn) Close() error {
	c.quitch <- struct{}{}
	return c.conn.CloseNow()
}

func (c *wsConn) readLoop(s WSServer) {
	for {
		select {
		case <-c.quitch:
			return
		default:
			ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
			defer cancel()

			t, m, err := c.conn.Read(ctx)
			if err == io.EOF {
				_ = c.Close()
				return
			}

			if err != nil {
				continue
			}

			if t != websocket.MessageText {
				color.Red("Unsupported message type, please send string")
				continue
			}

			s.handleMessage(c, m)
		}
	}
}
