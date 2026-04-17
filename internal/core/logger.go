package core

import (
	"fmt"
	"strings"
	"sync"
	"time"
)

type LogEntry struct {
	Timestamp time.Time `json:"timestamp"`
	Level     string    `json:"level"`
	Message   string    `json:"message"`
}

type LogManager struct {
	mu    sync.RWMutex
	logs  []LogEntry
	limit int
}

func NewLogManager(limit int) *LogManager {
	if limit <= 0 {
		limit = 1000
	}
	return &LogManager{
		logs:  make([]LogEntry, 0, limit),
		limit: limit,
	}
}

func (m *LogManager) addLog(level, format string, args ...interface{}) {
	m.mu.Lock()
	defer m.mu.Unlock()

	msg := fmt.Sprintf(format, args...)
	entry := LogEntry{
		Timestamp: time.Now(),
		Level:     level,
		Message:   strings.TrimSpace(msg),
	}

	m.logs = append(m.logs, entry)
	if len(m.logs) > m.limit {
		m.logs = m.logs[1:] // Keep within limit
	}

	// Also print to console for daemon mode
	fmt.Printf("[%s] [%s] %s\n", entry.Timestamp.Format("2006-01-02 15:04:05"), level, entry.Message)
}

func (m *LogManager) Info(format string, args ...interface{}) {
	m.addLog("INFO", format, args...)
}

func (m *LogManager) Error(format string, args ...interface{}) {
	m.addLog("ERROR", format, args...)
}

func (m *LogManager) Infof(format string, args ...interface{}) {
	m.addLog("INFO", format, args...)
}

func (m *LogManager) Errorf(format string, args ...interface{}) {
	m.addLog("ERROR", format, args...)
}

func (m *LogManager) GetLogs(limit int) []LogEntry {
	m.mu.RLock()
	defer m.mu.RUnlock()

	n := len(m.logs)
	if limit > 0 && limit < n {
		n = limit
	}

	result := make([]LogEntry, n)
	copy(result, m.logs[len(m.logs)-n:])
	return result
}
