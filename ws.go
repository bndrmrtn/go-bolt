package bolt

import (
	"net/http"
	"strings"
	"time"

	"github.com/coder/websocket"
)

func wsHandler(fn WSHandlerFunc) HandlerFunc {
	return func(c Ctx) error {
		if strings.ToLower(c.Header().Get("Upgrade")) != "websocket" {
			return NewError(http.StatusBadRequest, "websocket: request is not a websocket handshake")
		}

		w, r := c.ResponseWriter(), c.Request()

		conn, err := websocket.Accept(w, r, nil)
		if err != nil {
			return NewError(http.StatusInternalServerError, "websocket: failed to accept connection")
		}
		defer conn.CloseNow()

		serverLogger(time.Now(), "WS", string(c.IP()))
		fn(conn)
		return nil
	}
}
