package hub

import (
	"sync"

	"github.com/jackc/pgx/v5/pgtype"
)

type Client struct {
	send chan []byte
}

// defines the shape of hub 'obj'
type Hub struct {
	clients map[*Client]bool
	mu      sync.Mutex
}

// creates a Hub instance
func newHub() *Hub {
	return &Hub{
		clients: make(map[*Client]bool),
	}
}

// adds client to the hub
func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.clients[c] = true
}

// removes a client from the hub
func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	delete(h.clients, c)
	close(c.send)
}

// hub broadcast method
func (h *Hub) broadcast(msg []byte) {
	h.mu.Lock()
	defer h.mu.Unlock()
	for c := range h.clients {
		c.send <- msg
	}
}

// defines a Manager
type Manager struct {
	hubs map[pgtype.UUID]*Hub
	mu   sync.Mutex
}

// creates NewManager instance
func NewManager() *Manager {
	return &Manager{
		hubs: make(map[pgtype.UUID]*Hub),
	}
}

// creates a Hub instance or fetches it if it already exists
func (m *Manager) GetOrCreate(battleID pgtype.UUID) *Hub {
	m.mu.Lock()
	defer m.mu.Unlock()
	if h, ok := m.hubs[battleID]; ok {
		return h
	}
	h := newHub()
	m.hubs[battleID] = h
	return h
}

// sends a message to every client in the hub (defined by the battleID)
func (m *Manager) Broadcast(battleID pgtype.UUID, msg []byte) {
	m.mu.Lock()
	h, ok := m.hubs[battleID]
	m.mu.Unlock()
	if ok {
		h.broadcast(msg)
	}
}
