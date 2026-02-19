package room

import (
	"fmt"
	"sync"

	"github.com/google/uuid"
)

// Manager handles room lifecycle — creation, lookup, cleanup.
type Manager struct {
	mu    sync.RWMutex
	rooms map[string]*Room
}

func NewManager() *Manager {
	return &Manager{
		rooms: make(map[string]*Room),
	}
}

func (m *Manager) Create(roomType RoomType, tier int) (*Room, error) {
	tc, ok := Tiers[tier]
	if !ok {
		return nil, fmt.Errorf("unknown tier: %d", tier)
	}
	id := uuid.New().String()
	r := NewRoom(id, roomType, tc)

	m.mu.Lock()
	m.rooms[id] = r
	m.mu.Unlock()

	return r, nil
}

func (m *Manager) Get(id string) (*Room, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	r, ok := m.rooms[id]
	return r, ok
}

func (m *Manager) Remove(id string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.rooms, id)
}

func (m *Manager) ListByState(state RoomState) []*Room {
	m.mu.RLock()
	defer m.mu.RUnlock()
	var out []*Room
	for _, r := range m.rooms {
		if r.State == state {
			out = append(out, r)
		}
	}
	return out
}

func (m *Manager) Count() int {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return len(m.rooms)
}
