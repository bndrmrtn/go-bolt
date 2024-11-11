package gale

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"sync"

	"github.com/coder/websocket"
	"github.com/fatih/color"
)

type WSMessageHandler func(s WSServer, conn WSConn, msg []byte) error

// WSServer is the interface that wraps the basic methods for a websocket server.
type WSServer interface {
	// AddConnection adds a new connection to the server.
	AddConn(conn WSConn)
	// RemoveConnection removes a connection from the server.
	RemoveConn(conn WSConn)
	// RemoveConnection removes a connection from the server.
	Broadcast(msg []byte)
	// BroadcastFunc sends a message to all connections that satisfy the condition.
	BroadcastFunc(msg []byte, fn func(conn WSConn) bool)
	// BroadcastTo sends a message to a specific connection.
	BroadcastTo(msg []byte, conns ...WSConn)

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

	readLoop(s WSServer, ctx context.Context)
}

type wsServer struct {
	ctx            context.Context
	conns          []WSConn
	messageHandler WSMessageHandler
}

func NewWSServer(ctx context.Context) WSServer {
	s := &wsServer{
		ctx:   ctx,
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
	go conn.readLoop(s, s.ctx)
	s.conns = append(s.conns, conn)
}

func (s *wsServer) RemoveConn(conn WSConn) {
	for i, c := range s.conns {
		if c == conn {
			s.conns = append(s.conns[:i], s.conns[i+1:]...)
			break
		}
	}
}

func (s *wsServer) Broadcast(msg []byte) {
	for _, conn := range s.conns {
		conn.Conn().Write(s.ctx, websocket.MessageText, msg)
	}
}

func (s *wsServer) BroadcastFunc(msg []byte, fn func(conn WSConn) bool) {
	for _, conn := range s.conns {
		if fn(conn) {
			conn.Conn().Write(s.ctx, websocket.MessageText, msg)
		}
	}
}

func (s *wsServer) BroadcastTo(msg []byte, conns ...WSConn) {
	for _, conn := range conns {
		conn.Conn().Write(s.ctx, websocket.MessageText, msg)
	}
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
		ctx:    ctx,
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

func (c *wsConn) readLoop(s WSServer, ctx context.Context) {
	for {
		select {
		case <-c.quitch:
			break
		default:
			t, m, err := c.conn.Read(ctx)
			if err == io.EOF {
				_ = c.Close()
				break
			}

			if err != nil {
				continue
			}

			if t != websocket.MessageText {
				color.Red("Unsupported message type, please send string")
				continue
			}

			go s.handleMessage(c, m)
		}
	}
}
