package websocket

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

// Message types
const (
	MessageTypeJobStatus      = "job_status"
	MessageTypeJobLogs        = "job_logs"
	MessageTypeGPUMetrics     = "gpu_metrics"
	MessageTypeNotification   = "notification"
	MessageTypeBillingUpdate  = "billing_update"
	MessageTypeProviderStatus = "provider_status"
)

// Message represents a WebSocket message
type Message struct {
	Type      string                 `json:"type"`
	Data      map[string]interface{} `json:"data"`
	Timestamp time.Time              `json:"timestamp"`
}

// Client represents a WebSocket client
type Client struct {
	ID       string
	UserID   string
	Hub      *Hub
	Conn     *websocket.Conn
	Send     chan *Message
	mu       sync.Mutex
	isClosed bool
}

// Hub maintains active clients and broadcasts messages
type Hub struct {
	clients    map[string]*Client
	broadcast  chan *Message
	register   chan *Client
	unregister chan *Client
	mu         sync.RWMutex
}

// NewHub creates a new Hub
func NewHub() *Hub {
	return &Hub{
		clients:    make(map[string]*Client),
		broadcast:  make(chan *Message, 256),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

// Run starts the hub
func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client.ID] = client
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client.ID]; ok {
				delete(h.clients, client.ID)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for _, client := range h.clients {
				select {
				case client.Send <- message:
				default:
					close(client.Send)
					delete(h.clients, client.ID)
				}
			}
			h.mu.RUnlock()
		}
	}
}

// BroadcastToUser sends message to specific user
func (h *Hub) BroadcastToUser(userID string, message *Message) {
	h.mu.RLock()
	defer h.mu.RUnlock()

	for _, client := range h.clients {
		if client.UserID == userID {
			select {
			case client.Send <- message:
			default:
			}
		}
	}
}

// BroadcastToAll sends message to all connected clients
func (h *Hub) BroadcastToAll(message *Message) {
	h.broadcast <- message
}

// GetClientCount returns number of connected clients
func (h *Hub) GetClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

// GetUserClientCount returns number of clients for a user
func (h *Hub) GetUserClientCount(userID string) int {
	h.mu.RLock()
	defer h.mu.RUnlock()

	count := 0
	for _, client := range h.clients {
		if client.UserID == userID {
			count++
		}
	}
	return count
}

// ReadPump pumps messages from the websocket connection to the hub
func (c *Client) ReadPump() {
	defer func() {
		c.Hub.unregister <- c
		c.Conn.Close()
	}()

	c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.Conn.SetPongHandler(func(string) error {
		c.Conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.Conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
				// Log error
			}
			break
		}

		// Handle incoming messages (e.g., subscriptions)
		var msg Message
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		// Process message based on type
		c.handleMessage(&msg)
	}
}

// WritePump pumps messages from the hub to the websocket connection
func (c *Client) WritePump() {
	ticker := time.NewTicker(54 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.Conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}

			data, _ := json.Marshal(message)
			w.Write(data)

			// Add queued messages to the current websocket message
			n := len(c.Send)
			for i := 0; i < n; i++ {
				msg := <-c.Send
				data, _ := json.Marshal(msg)
				w.Write([]byte{'\n'})
				w.Write(data)
			}

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.Conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

// handleMessage handles incoming messages from client
func (c *Client) handleMessage(msg *Message) {
	switch msg.Type {
	case "subscribe":
		// Handle subscription to specific channels
		// e.g., subscribe to job updates, GPU metrics, etc.
	case "unsubscribe":
		// Handle unsubscription
	case "ping":
		// Respond with pong
		c.Send <- &Message{
			Type:      "pong",
			Timestamp: time.Now(),
		}
	}
}

// SendJobStatus sends job status update
func (h *Hub) SendJobStatus(userID, jobID, status string, data map[string]interface{}) {
	message := &Message{
		Type: MessageTypeJobStatus,
		Data: map[string]interface{}{
			"job_id": jobID,
			"status": status,
			"data":   data,
		},
		Timestamp: time.Now(),
	}
	h.BroadcastToUser(userID, message)
}

// SendJobLogs sends job logs
func (h *Hub) SendJobLogs(userID, jobID string, logs []string) {
	message := &Message{
		Type: MessageTypeJobLogs,
		Data: map[string]interface{}{
			"job_id": jobID,
			"logs":   logs,
		},
		Timestamp: time.Now(),
	}
	h.BroadcastToUser(userID, message)
}

// SendGPUMetrics sends GPU metrics
func (h *Hub) SendGPUMetrics(userID string, metrics map[string]interface{}) {
	message := &Message{
		Type:      MessageTypeGPUMetrics,
		Data:      metrics,
		Timestamp: time.Now(),
	}
	h.BroadcastToUser(userID, message)
}

// SendNotification sends notification
func (h *Hub) SendNotification(userID string, notification map[string]interface{}) {
	message := &Message{
		Type:      MessageTypeNotification,
		Data:      notification,
		Timestamp: time.Now(),
	}
	h.BroadcastToUser(userID, message)
}

// SendBillingUpdate sends billing update
func (h *Hub) SendBillingUpdate(userID string, billing map[string]interface{}) {
	message := &Message{
		Type:      MessageTypeBillingUpdate,
		Data:      billing,
		Timestamp: time.Now(),
	}
	h.BroadcastToUser(userID, message)
}

// SendProviderStatus sends provider status update
func (h *Hub) SendProviderStatus(providerID string, status map[string]interface{}) {
	message := &Message{
		Type:      MessageTypeProviderStatus,
		Data:      status,
		Timestamp: time.Now(),
	}
	// Broadcast to all clients interested in this provider
	h.BroadcastToAll(message)
}

