package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"net"
	"net/http"
	"net/url"
	"strings"
	"sync"
	"time"

	"github.com/gorilla/websocket"
	"github.com/rclone-backup-web/agent/logging"
)

var (
	ErrWebSocketClosed   = errors.New("hub websocket closed")
	ErrWebSocketNotReady = errors.New("hub websocket not connected")
)

type HubWSConn struct {
	ws *websocket.Conn

	send      chan []byte
	recv      chan WSMessage
	closed    chan struct{}
	closeOnce sync.Once
}

func DialHubWebSocket(baseURL, agentID, apiKey string) (*HubWSConn, error) {
	wsURL, origin, err := buildHubWSURL(baseURL)
	if err != nil {
		return nil, err
	}

	logging.Debugf("[HubWS] websocket dial url=%s origin=%s auth=%t", wsURL, strings.TrimSpace(origin), strings.TrimSpace(agentID) != "" && strings.TrimSpace(apiKey) != "")

	header := make(http.Header)
	if apiKey != "" && agentID != "" {
		header.Set("Authorization", fmt.Sprintf("Bearer %s:%s", agentID, apiKey))
	}
	if strings.TrimSpace(origin) != "" {
		header.Set("Origin", origin)
	}

	dialer := websocket.Dialer{
		Proxy:            http.ProxyFromEnvironment,
		HandshakeTimeout: 10 * time.Second,
		NetDialContext: (&net.Dialer{
			Timeout: 10 * time.Second,
		}).DialContext,
	}

	ctx, cancel := context.WithTimeout(context.Background(), 15*time.Second)
	defer cancel()

	ws, resp, err := dialer.DialContext(ctx, wsURL, header)
	if resp != nil && resp.Body != nil {
		logging.Debugf("[HubWS] websocket handshake status=%s", resp.Status)
		_ = resp.Body.Close()
	}
	if err != nil {
		return nil, err
	}

	conn := &HubWSConn{
		ws:     ws,
		send:   make(chan []byte, 512),
		recv:   make(chan WSMessage, 2048), // Increased from 256 to 2048 to prevent message drops
		closed: make(chan struct{}),
	}

	go conn.readLoop()
	go conn.writeLoop()

	return conn, nil
}

func (c *HubWSConn) Incoming() <-chan WSMessage {
	return c.recv
}

func (c *HubWSConn) IsClosed() bool {
	select {
	case <-c.closed:
		return true
	default:
		return false
	}
}

func (c *HubWSConn) Close() {
	c.closeOnce.Do(func() {
		close(c.closed)
		_ = c.ws.Close()
	})
}

func (c *HubWSConn) Send(msg WSMessage, timeout time.Duration) error {
	if c.IsClosed() {
		return ErrWebSocketClosed
	}

	data, err := json.Marshal(msg)
	if err != nil {
		return err
	}

	if timeout <= 0 {
		select {
		case c.send <- data:
			return nil
		default:
			return ErrWebSocketNotReady
		}
	}

	timer := time.NewTimer(timeout)
	defer timer.Stop()

	select {
	case c.send <- data:
		return nil
	case <-timer.C:
		return ErrWebSocketNotReady
	case <-c.closed:
		return ErrWebSocketClosed
	}
}

func (c *HubWSConn) SendJSON(msgType string, payload interface{}, timeout time.Duration) error {
	data, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	return c.Send(WSMessage{
		Type: msgType,
		Data: data,
	}, timeout)
}

func (c *HubWSConn) readLoop() {
	defer func() {
		close(c.recv)
		c.Close()
	}()

	for {
		_, raw, err := c.ws.ReadMessage()
		if err != nil {
			if closeErr, ok := err.(*websocket.CloseError); ok {
				logging.Debugf("[HubWS] read loop closed code=%d text=%q", closeErr.Code, closeErr.Text)
			} else {
				logging.Debugf("[HubWS] read loop error: %v", err)
			}
			return
		}

		rawStr := strings.TrimSpace(string(raw))
		if rawStr == "" {
			continue
		}

		// DIAGNOSTIC: Log raw message received
		log.Printf("[HubWS-DIAG] raw message received len=%d content=%s", len(rawStr), rawStr)

		var msg WSMessage
		if err := json.Unmarshal([]byte(rawStr), &msg); err != nil {
			log.Printf("[HubWS-DIAG] failed to unmarshal: %v", err)
			logging.Debugf("[HubWS] failed to unmarshal websocket frame len=%d err=%v", len(rawStr), err)
			continue
		}

		// DIAGNOSTIC: Log parsed message type
		log.Printf("[HubWS-DIAG] parsed message type=%s data_len=%d", msg.Type, len(msg.Data))

		select {
		case c.recv <- msg:
			log.Printf("[HubWS-DIAG] message sent to recv channel type=%s", msg.Type)
		case <-c.closed:
			return
		default:
			// Drop if the consumer is too slow; heartbeats/actions will arrive again.
			log.Printf("[HubWS-DIAG] DROPPED message type=%s (recv channel full)", msg.Type)
			logging.Debugf("[HubWS] dropping message type=%s (consumer too slow)", strings.TrimSpace(msg.Type))
		}
	}
}

func (c *HubWSConn) writeLoop() {
	defer c.Close()

	for {
		select {
		case <-c.closed:
			return
		case data := <-c.send:
			_ = c.ws.SetWriteDeadline(time.Now().Add(15 * time.Second))
			if err := c.ws.WriteMessage(websocket.TextMessage, data); err != nil {
				logging.Debugf("[HubWS] write loop error: %v", err)
				return
			}
		}
	}
}

func buildHubWSURL(baseURL string) (wsURL string, origin string, err error) {
	baseURL = strings.TrimSpace(baseURL)
	if baseURL == "" {
		return "", "", fmt.Errorf("hub base url is empty")
	}

	u, err := url.Parse(baseURL)
	if err != nil {
		return "", "", fmt.Errorf("invalid hub base url: %w", err)
	}

	switch u.Scheme {
	case "http":
		u.Scheme = "ws"
	case "https":
		u.Scheme = "wss"
	case "ws", "wss":
		// ok
	default:
		return "", "", fmt.Errorf("unsupported hub url scheme: %s", u.Scheme)
	}

	u.Path = strings.TrimRight(u.Path, "/") + "/api/v1/agent/ws"
	wsURL = u.String()

	originURL := *u
	if originURL.Scheme == "ws" {
		originURL.Scheme = "http"
	}
	if originURL.Scheme == "wss" {
		originURL.Scheme = "https"
	}
	originURL.Path = "/"
	originURL.RawQuery = ""
	originURL.Fragment = ""
	origin = originURL.String()

	return wsURL, origin, nil
}
