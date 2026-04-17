package state

import (
	"encoding/json"
	"os"
	"sync"
)

type State struct {
	ProcessedUIDs map[string]uint32 `json:"processed_uids"` // key: source_id, value: max UID
}

type Manager struct {
	path  string
	state State
	mu    sync.RWMutex
}

func NewManager(path string) (*Manager, error) {
	m := &Manager{
		path: path,
		state: State{
			ProcessedUIDs: make(map[string]uint32),
		},
	}
	if err := m.load(); err != nil && !os.IsNotExist(err) {
		return nil, err
	}
	return m, nil
}

func (m *Manager) load() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	data, err := os.ReadFile(m.path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, &m.state)
}

func (m *Manager) save() error {
	data, err := json.MarshalIndent(m.state, "", "  ")
	if err != nil {
		return err
	}
	return os.WriteFile(m.path, data, 0644)
}

func (m *Manager) GetUID(sourceID string) uint32 {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.state.ProcessedUIDs[sourceID]
}

func (m *Manager) SetUID(sourceID string, uid uint32) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.state.ProcessedUIDs == nil {
		m.state.ProcessedUIDs = make(map[string]uint32)
	}

	// Only update if the new UID is greater
	if uid > m.state.ProcessedUIDs[sourceID] {
		m.state.ProcessedUIDs[sourceID] = uid
		return m.save()
	}
	return nil
}
