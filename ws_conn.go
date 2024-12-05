package gale

import (
	"context"
	"encoding/json"
	"time"

	"github.com/coder/websocket"
	"github.com/fatih/color"
)

// Interfaces

// WSConn is the interface that wraps the basic methods for a websocket connection.
type WSConn interface {
	ID() string

	Ctx() Ctx
	Conn() *websocket.Conn
	Close() error

	Send(data []byte) error
	SendJSON(v any) error

	readLoop(s WSServer)
}

// WSMessage is the interface that wraps the basic methods for a websocket message.
type WSMessage interface {
	Conn() WSConn
	Content() []byte
}

// Implementations

type socketMessage struct {
	content []byte
	conn    WSConn
}

func (m *socketMessage) Conn() WSConn {
	return m.conn
}

func (m *socketMessage) Content() []byte {
	return m.content
}

type socketConn struct {
	id string

	ctx  Ctx
	conn *websocket.Conn

	quitch chan struct{}
}

func NewWSConn(ctx Ctx, conn *websocket.Conn) WSConn {
	return &socketConn{
		id:     ctx.ID(),
		ctx:    ctx,
		conn:   conn,
		quitch: make(chan struct{}),
	}
}

func (c *socketConn) ID() string {
	return c.id
}

func (c *socketConn) Ctx() Ctx {
	return c.ctx
}

func (c *socketConn) Conn() *websocket.Conn {
	return c.conn
}

func (c *socketConn) Close() error {
	c.quitch <- struct{}{}
	return c.conn.Close(websocket.StatusGoingAway, "Closed by server.")
}

func (c *socketConn) Send(m []byte) error {
	return c.conn.Write(context.Background(), websocket.MessageText, m)
}

func (c *socketConn) SendJSON(v any) error {
	b, err := json.Marshal(v)
	if err != nil {
		return err
	}

	return c.Send(b)
}

func (c *socketConn) readLoop(s WSServer) {
	defer s.RemoveConn(c, true)

	for {
		select {
		case <-c.quitch:
			return
		default:
			var timeout time.Duration

			wsConf := s.Config()

			if wsConf != nil {
				timeout = wsConf.ReadTimeout
			} else {
				timeout = time.Second * 10
			}

			ctx, cancel := context.WithTimeout(context.Background(), timeout)

			t, m, err := c.conn.Read(ctx)

			cancel()
			if err != nil {
				_ = c.Close()
				return
			}

			if t != websocket.MessageText {
				color.Red("Unsupported message type, please send string")
				continue
			}

			s.receive(&socketMessage{
				content: m,
				conn:    c,
			})
		}
	}
}
