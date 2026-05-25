package services

import (
	"encoding/json"
	"errors"
	"log"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/websocket"
)

var (
	ErrAgentNotConnected = errors.New("agent websocket not connected")
	ErrSendQueueFull     = errors.New("agent websocket send queue full")
)

type AgentWSService struct {
	mu    sync.RWMutex
	conns map[uuid.UUID]*AgentWSConn
}

type AgentWSConn struct {
	agentID uuid.UUID
	ws      *websocket.Conn

	send      chan []byte
	closed    chan struct{}
	closeOnce sync.Once
}

func NewAgentWSService() *AgentWSService {
	return &AgentWSService{
		conns: make(map[uuid.UUID]*AgentWSConn),
	}
}

func (s *AgentWSService) Register(agentID uuid.UUID, ws *websocket.Conn) *AgentWSConn {
	conn := &AgentWSConn{
		agentID: agentID,
		ws:      ws,
		send:    make(chan []byte, 2048), // Increased from 256 to 2048 to prevent send queue full errors
		closed:  make(chan struct{}),
	}

	var previous *AgentWSConn
	s.mu.Lock()
	previous = s.conns[agentID]
	s.conns[agentID] = conn
	s.mu.Unlock()

	if previous != nil {
		previous.Close()
	}

	log.Printf("[AgentWS] connected agent=%s", agentID)
	go conn.writeLoop()

	return conn
}

func (s *AgentWSService) Unregister(agentID uuid.UUID) {
	var conn *AgentWSConn
	s.mu.Lock()
	conn = s.conns[agentID]
	delete(s.conns, agentID)
	s.mu.Unlock()
	if conn != nil {
		conn.Close()
	}
}

// UnregisterConn removes an agent connection only if it matches the currently registered one.
// This prevents an old connection's cleanup from disconnecting a newer replacement connection.
func (s *AgentWSService) UnregisterConn(agentID uuid.UUID, conn *AgentWSConn) {
	if conn == nil {
		return
	}

	s.mu.Lock()
	current := s.conns[agentID]
	if current != conn {
		s.mu.Unlock()
		return
	}
	delete(s.conns, agentID)
	s.mu.Unlock()

	conn.Close()
}

func (s *AgentWSService) IsConnected(agentID uuid.UUID) bool {
	s.mu.RLock()
	conn := s.conns[agentID]
	s.mu.RUnlock()
	return conn != nil && !conn.IsClosed()
}

func (s *AgentWSService) SendJSON(agentID uuid.UUID, payload interface{}) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}
	return s.SendBytes(agentID, data)
}

func (s *AgentWSService) SendBytes(agentID uuid.UUID, data []byte) error {
	s.mu.RLock()
	conn := s.conns[agentID]
	s.mu.RUnlock()
	if conn == nil || conn.IsClosed() {
		return ErrAgentNotConnected
	}
	return conn.Send(data, 3*time.Second)
}

// SendBytesWithTimeout sends data to an agent with a custom timeout
func (s *AgentWSService) SendBytesWithTimeout(agentID uuid.UUID, data []byte, timeout time.Duration) error {
	s.mu.RLock()
	conn := s.conns[agentID]
	s.mu.RUnlock()
	if conn == nil || conn.IsClosed() {
		return ErrAgentNotConnected
	}
	return conn.Send(data, timeout)
}

func (c *AgentWSConn) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *AgentWSConn) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.ws.Close()
		log.Printf("[AgentWS] disconnected agent=%s", c.agentID)
	})
}

func (c *AgentWSConn) Send(data []byte, timeout time.Duration) error {
	if c.IsClosed() {
		return ErrAgentNotConnected
	}

	if timeout <= 0 {
		select {
		case c.send <- data:
			return nil
		case <-c.closed:
			return ErrAgentNotConnected
		default:
			return ErrSendQueueFull
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c.send <- data:
		return nil
	case <-timer.C:
		return ErrSendQueueFull
	case <-c.closed:
		return ErrAgentNotConnected
	}
}

func (c *AgentWSConn) writeLoop() {
	for {
		select {
		case <-c.closed:
			return
		case data, ok := <-c.send:
			if !ok {
				return
			}

			_ = c.ws.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
				log.Printf("[AgentWS] write failed agent=%s: %v", c.agentID, err)
				c.Close()
				return
			}
		}
	}
}
