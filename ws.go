package bolt

import (
	"log"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func wsHandler(fn WSHandlerFunc) HandlerFunc {
	return func(c Ctx) error {
		if strings.ToLower(c.Header().Get("Upgrade")) != "websocket" {
			log.Println("websocket: request is not a websocket handshake")
			return nil
		}

		w, r := c.ResponseWriter(), c.Request()

		conn, err := websocket.Accept(w, r, c.App().config.Websocket.AcceptOptions)
		if err != nil {
			log.Println("websocket: failed to accept connection", err)
			return nil
		}

		serverLogger(time.Now(), "WS", string(c.IP()))
		fn(newWSConn(c, conn))
		return nil
	}
}
