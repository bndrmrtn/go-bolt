package gale

import (
	"log"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func wsHandler(fn WSHandlerFunc) HandlerFunc {
	return func(c Ctx) error {
		start := time.Now()

		if strings.ToLower(c.Header().Get("Upgrade")) != "websocket" {
			log.Println("websocket: request is not a websocket handshake")
			return nil
		}

		w, r := c.ResponseWriter(), c.Request()

		conn, err := websocket.Accept(w, r, c.App().config.WebSocket)
		if err != nil {
			log.Println("websocket: failed to accept connection: ", err)
			return nil
		}

		serverLogger(start, "WS", c.IP())

		newConn := NewWSConn(c, conn)
		fn(newConn)

		return nil
	}
}
