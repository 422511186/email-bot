package core

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
)

// sourceState 追踪每个邮箱的进度。
type sourceState struct {
	LastUID     uint32 `json:"last_uid"`
	Initialized bool   `json:"initialized"` // 完成首次运行扫描后为 true
}

// State 保存所有跨重启的持久化数据。
type State struct {
	mu      sync.RWMutex
	Sources map[string]*sourceState `json:"sources"`
}

// NewState 返回空状态。
func NewState() *State {
	return &State{Sources: make(map[string]*sourceState)}
}

// LoadState 从磁盘读取状态；任何错误均返回 NewState()。
func LoadState(path string) (*State, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取状态文件: %w", err)
	}
	s := NewState()
	if err := json.Unmarshal(data, s); err != nil {
		return nil, fmt.Errorf("解析状态: %w", err)
	}
	return s, nil
}

// Save 以原子方式将状态写入磁盘。
func (s *State) Save(path string) error {
	s.mu.RLock()
	data, err := json.MarshalIndent(s, "", "  ")
	s.mu.RUnlock()
	if err != nil {
		return err
	}

	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		return err
	}

	// 先写入临时文件，然后重命名以保证原子性。
	tmp := path + ".tmp"
	if err := os.WriteFile(tmp, data, 0o600); err != nil {
		return err
	}
	return os.Rename(tmp, path)
}

// GetLastUID 返回指定用户名的最后处理的 UID。
func (s *State) GetLastUID(username string) uint32 {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if src, ok := s.Sources[username]; ok {
		return src.LastUID
	}
	return 0
}

// IsInitialized 返回该邮箱是否已完成首次运行设置。
func (s *State) IsInitialized(username string) bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if src, ok := s.Sources[username]; ok {
		return src.Initialized
	}
	return false
}

// SetLastUID 记录最近处理的 UID。
func (s *State) SetLastUID(username string, uid uint32) {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.Sources == nil {
		s.Sources = make(map[string]*sourceState)
	}
	if _, ok := s.Sources[username]; !ok {
		s.Sources[username] = &sourceState{}
	}
	s.Sources[username].LastUID = uid
	s.Sources[username].Initialized = true
}
