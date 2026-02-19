package server

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"sync"
	"time"

	"github.com/coder/websocket"
	"github.com/coder/websocket/wsjson"
	"github.com/lastclick/lastclick/internal/auth"
)

// WSMessage is the envelope for all WebSocket communication.
type WSMessage struct {
	Type    string          `json:"type"`
	Payload json.RawMessage `json:"payload,omitempty"`
}

// Client represents a connected Mini App player.
type Client struct {
	ID     int64
	RoomID string
	conn   *websocket.Conn
	send   chan WSMessage
}

// Hub manages all WebSocket clients and room-level broadcasting.
type Hub struct {
	mu       sync.RWMutex
	clients  map[int64]*Client
	rooms    map[string]map[int64]*Client
	handler  MessageHandler
	botToken string
	logger   *slog.Logger
}

// MessageHandler processes inbound messages from a client.
type MessageHandler interface {
	HandleMessage(ctx context.Context, client *Client, msg WSMessage)
}

func NewHub(botToken string, handler MessageHandler, logger *slog.Logger) *Hub {
	return &Hub{
		clients:  make(map[int64]*Client),
		rooms:    make(map[string]map[int64]*Client),
		handler:  handler,
		botToken: botToken,
		logger:   logger,
	}
}

func (h *Hub) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	initData := r.URL.Query().Get("initData")
	if err := auth.ValidateInitData(initData, h.botToken); err != nil {
		http.Error(w, "unauthorized", http.StatusUnauthorized)
		return
	}

	userID, err := extractUserID(initData)
	if err != nil {
		http.Error(w, "bad init data", http.StatusBadRequest)
		return
	}

	conn, err := websocket.Accept(w, r, &websocket.AcceptOptions{
		InsecureSkipVerify: true,
	})
	if err != nil {
		h.logger.Error("ws accept", "err", err)
		return
	}

	client := &Client{
		ID:   userID,
		conn: conn,
		send: make(chan WSMessage, 64),
	}

	h.register(client)
	defer h.unregister(client)

	ctx, cancel := context.WithCancel(r.Context())
	defer cancel()

	go h.writePump(ctx, client)
	h.readPump(ctx, client)
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c.ID] = c
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if _, ok := h.clients[c.ID]; ok {
		delete(h.clients, c.ID)
		close(c.send)
	}
	if c.RoomID != "" {
		if room, ok := h.rooms[c.RoomID]; ok {
			delete(room, c.ID)
			if len(room) == 0 {
				delete(h.rooms, c.RoomID)
			}
		}
	}
}

// JoinRoom adds a client to a room broadcast group.
func (h *Hub) JoinRoom(clientID int64, roomID string) {
	h.mu.Lock()
	defer h.mu.Unlock()
	c, ok := h.clients[clientID]
	if !ok {
		return
	}
	if c.RoomID != "" && c.RoomID != roomID {
		if room, ok := h.rooms[c.RoomID]; ok {
			delete(room, c.ID)
		}
	}
	c.RoomID = roomID
	if _, ok := h.rooms[roomID]; !ok {
		h.rooms[roomID] = make(map[int64]*Client)
	}
	h.rooms[roomID][c.ID] = c
}

// BroadcastRoom sends a message to every client in a room.
func (h *Hub) BroadcastRoom(roomID string, msg WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	room, ok := h.rooms[roomID]
	if !ok {
		return
	}
	for _, c := range room {
		select {
		case c.send <- msg:
		default:
			h.logger.Warn("client send buffer full", "client", c.ID)
		}
	}
}

// SendTo sends a message to a specific client.
func (h *Hub) SendTo(clientID int64, msg WSMessage) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[clientID]
	if !ok {
		return
	}
	select {
	case c.send <- msg:
	default:
	}
}

// GetClient returns a client by ID.
func (h *Hub) GetClient(clientID int64) (*Client, bool) {
	h.mu.RLock()
	defer h.mu.RUnlock()
	c, ok := h.clients[clientID]
	return c, ok
}

func (h *Hub) readPump(ctx context.Context, c *Client) {
	defer func() {
		if err := c.conn.CloseNow(); err != nil {
			h.logger.Error("close conn", "err", err)
		}
	}()
	for {
		var msg WSMessage
		if err := wsjson.Read(ctx, c.conn, &msg); err != nil {
			return
		}
		if h.handler != nil {
			h.handler.HandleMessage(ctx, c, msg)
		}
	}
}

func (h *Hub) writePump(ctx context.Context, c *Client) {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for {
		select {
		case msg, ok := <-c.send:
			if !ok {
				c.conn.Close(websocket.StatusNormalClosure, "")
				return
			}
			if err := wsjson.Write(ctx, c.conn, msg); err != nil {
				return
			}
		case <-ticker.C:
			if err := c.conn.Ping(ctx); err != nil {
				return
			}
		case <-ctx.Done():
			return
		}
	}
}

func extractUserID(initData string) (int64, error) {
	vals, err := url.ParseQuery(initData)
	if err != nil {
		return 0, err
	}

	userJSON := vals.Get("user")
	if userJSON == "" {
		return 0, fmt.Errorf("missing user")
	}

	var u struct {
		ID int64 `json:"id"`
	}
	if err := json.Unmarshal([]byte(userJSON), &u); err != nil {
		return 0, err
	}
	if u.ID == 0 {
		return 0, fmt.Errorf("invalid user id")
	}
	return u.ID, nil
}
