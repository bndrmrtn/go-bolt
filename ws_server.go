package gale

import (
	"sync"
)

// Interfaces

// WSDispatcher is a function that handles incoming messages.
type WSDispatcher func(s WSServer, msg WSMessage) error

// WSServer is the interface that wraps the basic methods for a websocket server.
type WSServer interface {
	Config() *WSConfig

	// AddConnection adds a new connection to the server.
	AddConn(conn WSConn)
	// RemoveConnection removes a connection from the server.
	RemoveConn(conn WSConn, close ...bool) error
	// RemoveConnection removes a connection from the server.
	Broadcast(msg []byte)
	// BroadcastFunc sends a message to all connections that satisfy the condition.
	BroadcastFunc(msg []byte, fn func(conn WSConn) bool)
	// BroadcastTo sends a message to a specific connection.
	BroadcastTo(msg []byte, conns ...WSConn)

	// Close closes the server.
	Close() error

	handleReceiveLoop()
	handleSendLoop()
	receive(m WSMessage)
}

// Implementations

type broadcastMessage struct {
	content []byte
	fn      func(conn WSConn) bool
}

func (b *broadcastMessage) can(conn WSConn) bool {
	if b.fn == nil {
		return true
	}
	return b.fn(conn)
}

type socketServer struct {
	conf *WSConfig

	conns      map[string]WSConn
	dispatcher WSDispatcher

	quitch   chan struct{}
	msgInCh  chan WSMessage
	msgOutCh chan *broadcastMessage

	mu sync.RWMutex
}

// NewWebSocketServer creates a new websocket server.
func NewWebSocketServer(dispatcher WSDispatcher, conf ...*WSConfig) WSServer {
	if len(conf) == 0 {
		conf = append(conf, defaultWSConfig())
	}

	config := conf[0]
	config.check()

	s := &socketServer{
		conf: config,

		conns:      make(map[string]WSConn),
		dispatcher: dispatcher,

		quitch:   make(chan struct{}),
		msgInCh:  make(chan WSMessage, config.MessageBufferSize),
		msgOutCh: make(chan *broadcastMessage, config.MessageBufferSize),
	}

	go s.handleReceiveLoop()
	go s.handleSendLoop()

	return s
}

func (s *socketServer) Config() *WSConfig {
	return s.conf
}

func (s *socketServer) AddConn(conn WSConn) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.conns[conn.ID()] = conn
	go conn.readLoop(s)
}

func (s *socketServer) RemoveConn(conn WSConn, close ...bool) error {
	var closeConn bool
	if len(close) == 0 || close[0] {
		closeConn = close[0]
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	if closeConn {
		if err := conn.Close(); err != nil {
			return err
		}
	}

	delete(s.conns, conn.ID())
	return nil
}

func (s *socketServer) Broadcast(content []byte) {
	s.msgOutCh <- &broadcastMessage{
		content: content,
	}
}

func (s *socketServer) BroadcastFunc(content []byte, fn func(conn WSConn) bool) {
	s.msgOutCh <- &broadcastMessage{
		content: content,
		fn:      fn,
	}
}

func (s *socketServer) BroadcastTo(content []byte, conns ...WSConn) {
	s.BroadcastFunc(content, func(conn WSConn) bool {
		for _, c := range conns {
			if conn.ID() == c.ID() {
				return true
			}
		}
		return false
	})
}

func (s *socketServer) Close() error {
	s.quitch <- struct{}{}

	s.mu.Lock()
	for _, conn := range s.conns {
		s.RemoveConn(conn, true)
	}
	s.mu.Unlock()

	return nil
}

func (s *socketServer) handleReceiveLoop() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.conf.MaxConcurrentReads)

	for {
		select {
		case msg := <-s.msgInCh:
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				if err := s.dispatcher(s, msg); err != nil {
					_ = msg.Conn().Close()
				}
			}()
		case <-s.quitch:
			wg.Wait()
			return
		}
	}
}

func (s *socketServer) handleSendLoop() {
	var wg sync.WaitGroup
	sem := make(chan struct{}, s.conf.MaxConcurrentWrites)

	for {
		select {
		case msg := <-s.msgOutCh:
			wg.Add(1)
			sem <- struct{}{}
			go func() {
				defer wg.Done()
				defer func() { <-sem }()
				s.handleSend(msg)
			}()
		case <-s.quitch:
			wg.Wait()
			return
		}
	}
}

func (s *socketServer) handleSend(msg *broadcastMessage) {
	s.mu.RLock()
	for _, conn := range s.conns {
		if msg.can(conn) {
			_ = conn.Send(msg.content)
		}
	}
	s.mu.RUnlock()
}

func (s *socketServer) receive(m WSMessage) {
	s.msgInCh <- m
}
