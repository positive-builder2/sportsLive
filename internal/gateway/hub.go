package gateway

import (
	"encoding/json"
	"sync"
)

type Client struct {
	matchID string
	send    chan []byte
}

type Hub struct {
	mu         sync.RWMutex
	rooms      map[string]map[*Client]struct{}
	register   chan *Client
	unregister chan *Client
}

func NewHub() *Hub {
	return &Hub{
		rooms:      make(map[string]map[*Client]struct{}),
		register:   make(chan *Client),
		unregister: make(chan *Client),
	}
}

func (h *Hub) Run() {
	for {
		select {
		case c := <-h.register:
			h.mu.Lock()
			if h.rooms[c.matchID] == nil {
				h.rooms[c.matchID] = make(map[*Client]struct{})
			}
			h.rooms[c.matchID][c] = struct{}{}
			h.mu.Unlock()
		case c := <-h.unregister:
			h.mu.Lock()
			if room, ok := h.rooms[c.matchID]; ok {
				if _, exists := room[c]; exists {
					delete(room, c)
					close(c.send)
					if len(room) == 0 {
						delete(h.rooms, c.matchID)
					}
				}
			}
			h.mu.Unlock()
		}
	}
}

func (h *Hub) Broadcast(matchID string, payload any) {
	data, err := json.Marshal(payload)
	if err != nil {
		return
	}

	h.mu.RLock()
	defer h.mu.RUnlock()

	// Send to match specific room
	if room, ok := h.rooms[matchID]; ok {
		for c := range room {
			select {
			case c.send <- data:
			default:
			}
		}
	}

	// Send to global "all" room if different
	if matchID != "all" {
		if room, ok := h.rooms["all"]; ok {
			for c := range room {
				select {
				case c.send <- data:
				default:
				}
			}
		}
	}
}

func (h *Hub) Register(c *Client)   { h.register <- c }
func (h *Hub) Unregister(c *Client) { h.unregister <- c }
