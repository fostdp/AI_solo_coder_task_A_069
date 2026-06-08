package ws

import (
	"context"
	"encoding/json"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gorilla/websocket"
)

type WSMessage struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data"`
}

type Client struct {
	ID   string
	Conn *websocket.Conn
	Send chan []byte
	Hub  *Hub
}

type Hub struct {
	clients    map[*Client]bool
	register   chan *Client
	unregister chan *Client
	broadcast  chan []byte
	mu         sync.RWMutex
	upgrader   websocket.Upgrader
}

func NewHub() *Hub {
	return &Hub{
		clients:    make(map[*Client]bool),
		register:   make(chan *Client),
		unregister: make(chan *Client),
		broadcast:  make(chan []byte),
		upgrader: websocket.Upgrader{
			CheckOrigin: func(r *http.Request) bool {
				return true
			},
		},
	}
}

func (h *Hub) Run(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			h.mu.Lock()
			for client := range h.clients {
				close(client.Send)
				delete(h.clients, client)
			}
			h.mu.Unlock()
			return
		case client := <-h.register:
			h.mu.Lock()
			h.clients[client] = true
			h.mu.Unlock()
		case client := <-h.unregister:
			h.mu.Lock()
			if _, ok := h.clients[client]; ok {
				delete(h.clients, client)
				close(client.Send)
			}
			h.mu.Unlock()
		case message := <-h.broadcast:
			h.mu.RLock()
			for client := range h.clients {
				select {
				case client.Send <- message:
				default:
					h.mu.RUnlock()
					h.mu.Lock()
					delete(h.clients, client)
					close(client.Send)
					h.mu.Unlock()
					h.mu.RLock()
				}
			}
			h.mu.RUnlock()
		}
	}
}

func (h *Hub) Register(client *Client) {
	h.register <- client
}

func (h *Hub) Unregister(client *Client) {
	h.unregister <- client
}

func (h *Hub) Broadcast(message *WSMessage) error {
	data, err := json.Marshal(message)
	if err != nil {
		return err
	}
	h.broadcast <- data
	return nil
}

func (h *Hub) BroadcastDeviceUpdate(data json.RawMessage) {
	_ = h.Broadcast(&WSMessage{Type: "device_update", Data: data})
}

func (h *Hub) BroadcastPUEUpdate(data json.RawMessage) {
	_ = h.Broadcast(&WSMessage{Type: "pue_update", Data: data})
}

func (h *Hub) BroadcastOptimization(data json.RawMessage) {
	_ = h.Broadcast(&WSMessage{Type: "optimization", Data: data})
}

func (h *Hub) BroadcastAlert(data json.RawMessage) {
	_ = h.Broadcast(&WSMessage{Type: "alert", Data: data})
}

func (h *Hub) ClientCount() int {
	h.mu.RLock()
	defer h.mu.RUnlock()
	return len(h.clients)
}

func (h *Hub) HandleWebSocket(w http.ResponseWriter, r *http.Request) {
	conn, err := h.upgrader.Upgrade(w, r, nil)
	if err != nil {
		log.Println("upgrade error:", err)
		return
	}
	client := NewClient(conn, h)
	h.Register(client)

	go client.writePump()
	go client.readPump()
}

func NewClient(conn *websocket.Conn, hub *Hub) *Client {
	return &Client{
		ID:   time.Now().Format("20060102150405.000000000"),
		Conn: conn,
		Send: make(chan []byte, 256),
		Hub:  hub,
	}
}

func (c *Client) readPump() {
	defer func() {
		c.Hub.Unregister(c)
		c.Conn.Close()
	}()

	c.Conn.SetPongHandler(func(string) error {
		return nil
	})

	for {
		_, _, err := c.Conn.ReadMessage()
		if err != nil {
			break
		}
	}
}

func (c *Client) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.Conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.Send:
			if !ok {
				_ = c.Conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}
			if err := c.Conn.WriteMessage(websocket.TextMessage, message); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.Conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}
