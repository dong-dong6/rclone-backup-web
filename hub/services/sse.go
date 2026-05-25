package services

import (
	"encoding/json"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SSEEvent struct {
	Type string                 `json:"type"`
	Data map[string]interface{} `json:"data"`
	Time time.Time              `json:"time"`
}

type SSEClient struct {
	ID       string
	Channel  chan SSEEvent
	Context  *gin.Context
	CloseSignal chan bool
}

type SSEService struct {
	clients    map[string]*SSEClient
	clientsMux sync.RWMutex
	broadcast  chan SSEEvent
}

func NewSSEService() *SSEService {
	return &SSEService{
		clients:   make(map[string]*SSEClient),
		broadcast: make(chan SSEEvent, 100),
	}
}

// Start starts the SSE service
func (s *SSEService) Start() {
	go s.broadcastLoop()
}

// broadcastLoop handles broadcasting events to all connected clients
func (s *SSEService) broadcastLoop() {
	for event := range s.broadcast {
		s.clientsMux.RLock()
		for _, client := range s.clients {
			select {
			case client.Channel <- event:
				// Event sent successfully
			case <-time.After(1 * time.Second):
				// Client is not responsive, skip
				log.Printf("Client %s is not responsive, skipping event", client.ID)
			}
		}
		s.clientsMux.RUnlock()
	}
}

// SendEvent sends an event to all connected clients
func (s *SSEService) SendEvent(eventType string, data map[string]interface{}) {
	event := SSEEvent{
		Type: eventType,
		Data: data,
		Time: time.Now(),
	}
	
	select {
	case s.broadcast <- event:
		// Event queued successfully
	case <-time.After(1 * time.Second):
		// Broadcast channel is full, drop event
		log.Printf("Broadcast channel is full, dropping event: %s", eventType)
	}
}

// AddClient adds a new SSE client
func (s *SSEService) AddClient(c *gin.Context) string {
	clientID := fmt.Sprintf("client-%d", time.Now().UnixNano())
	
	client := &SSEClient{
		ID:          clientID,
		Channel:     make(chan SSEEvent, 10),
		Context:     c,
		CloseSignal: make(chan bool),
	}
	
	s.clientsMux.Lock()
	s.clients[clientID] = client
	s.clientsMux.Unlock()
	
	log.Printf("SSE client connected: %s", clientID)
	
	// Send initial connection event
	client.Channel <- SSEEvent{
		Type: "connected",
		Data: map[string]interface{}{"client_id": clientID},
		Time: time.Now(),
	}
	
	return clientID
}

// RemoveClient removes an SSE client
func (s *SSEService) RemoveClient(clientID string) {
	s.clientsMux.Lock()
	defer s.clientsMux.Unlock()
	
	if client, exists := s.clients[clientID]; exists {
		close(client.Channel)
		close(client.CloseSignal)
		delete(s.clients, clientID)
		log.Printf("SSE client disconnected: %s", clientID)
	}
}

// GetClient gets a client by ID
func (s *SSEService) GetClient(clientID string) (*SSEClient, bool) {
	s.clientsMux.RLock()
	defer s.clientsMux.RUnlock()
	
	client, exists := s.clients[clientID]
	return client, exists
}

// StreamToClient streams events to a specific client
func (s *SSEService) StreamToClient(c *gin.Context, clientID string) {
	client, exists := s.GetClient(clientID)
	if !exists {
		return
	}
	
	c.Header("Content-Type", "text/event-stream")
	c.Header("Cache-Control", "no-cache")
	c.Header("Connection", "keep-alive")
	c.Header("X-Accel-Buffering", "no")
	
	// Send periodic heartbeats
	heartbeatTicker := time.NewTicker(30 * time.Second)
	defer heartbeatTicker.Stop()
	
	for {
		select {
		case event, ok := <-client.Channel:
			if !ok {
				return
			}
			
			data, _ := json.Marshal(event.Data)
			fmt.Fprintf(c.Writer, "event: %s\n", event.Type)
			fmt.Fprintf(c.Writer, "data: %s\n\n", data)
			c.Writer.Flush()
			
		case <-heartbeatTicker.C:
			fmt.Fprintf(c.Writer, ": heartbeat\n\n")
			c.Writer.Flush()
			
		case <-client.CloseSignal:
			return
			
		case <-c.Request.Context().Done():
			return
		}
	}
}