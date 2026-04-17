package tui

import (
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"time"

	"email-bot/internal/core"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"
)

var (
	baseStyle = lipgloss.NewStyle().
			BorderStyle(lipgloss.NormalBorder()).
			BorderForeground(lipgloss.Color("240"))

	titleStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("205")).
			Bold(true).
			MarginBottom(1)

	infoStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("86"))

	logStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("252"))
)

type model struct {
	apiAddress string
	logs       []core.LogEntry
	status     map[string]interface{}
	err        error
	width      int
	height     int
}

type tickMsg time.Time

func fetchAPI(addr string) tea.Msg {
	// Fetch Status
	respStatus, err := http.Get(fmt.Sprintf("http://%s/api/status", addr))
	var status map[string]interface{}
	if err == nil {
		defer respStatus.Body.Close()
		json.NewDecoder(respStatus.Body).Decode(&status)
	}

	// Fetch Logs
	respLogs, err := http.Get(fmt.Sprintf("http://%s/api/logs", addr))
	var logs []core.LogEntry
	if err == nil {
		defer respLogs.Body.Close()
		json.NewDecoder(respLogs.Body).Decode(&logs)
	}

	return apiDataMsg{
		status: status,
		logs:   logs,
		err:    err,
	}
}

type apiDataMsg struct {
	status map[string]interface{}
	logs   []core.LogEntry
	err    error
}

func initialModel(apiAddress string) model {
	return model{
		apiAddress: apiAddress,
		status:     make(map[string]interface{}),
	}
}

func (m model) Init() tea.Cmd {
	return tea.Batch(
		tea.Tick(time.Second*2, func(t time.Time) tea.Msg {
			return tickMsg(t)
		}),
		func() tea.Msg {
			return fetchAPI(m.apiAddress)
		},
	)
}

func (m model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.KeyMsg:
		if msg.String() == "q" || msg.String() == "ctrl+c" {
			return m, tea.Quit
		}
	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height
	case tickMsg:
		return m, tea.Batch(
			func() tea.Msg { return fetchAPI(m.apiAddress) },
			tea.Tick(time.Second*2, func(t time.Time) tea.Msg { return tickMsg(t) }),
		)
	case apiDataMsg:
		if msg.err != nil {
			m.err = msg.err
		} else {
			m.err = nil
			m.status = msg.status
			m.logs = msg.logs
		}
	}
	return m, nil
}

func (m model) View() string {
	if m.width == 0 {
		return "Initializing..."
	}

	var s strings.Builder

	s.WriteString(titleStyle.Render("🤖 Email Bot TUI Dashboard"))
	s.WriteString("\n")

	if m.err != nil {
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("9")).Render(fmt.Sprintf("Error connecting to Daemon: %v", m.err)))
		s.WriteString("\n\nPress 'q' to quit.")
		return baseStyle.Width(m.width - 2).Height(m.height - 2).Render(s.String())
	}

	// Status Section
	s.WriteString(infoStyle.Render(fmt.Sprintf("状态 (Status): %v", m.status["status"])))
	s.WriteString("\n")
	s.WriteString(fmt.Sprintf("监听源 (Sources): %v | 目标 (Targets): %v | 规则 (Rules): %v",
		m.status["sources_count"], m.status["targets_count"], m.status["rules_count"]))
	s.WriteString("\n\n")

	// Logs Section
	s.WriteString(titleStyle.Render("实时日志 (Live Logs):"))
	s.WriteString("\n")

	// Display bottom 10 logs or less
	logCount := len(m.logs)
	displayLimit := 15
	if logCount < displayLimit {
		displayLimit = logCount
	}

	for i := logCount - displayLimit; i < logCount; i++ {
		entry := m.logs[i]
		color := "252"
		if entry.Level == "ERROR" {
			color = "9" // Red
		} else if entry.Level == "INFO" {
			color = "86" // Cyan
		}
		
		line := fmt.Sprintf("[%s] %s", entry.Timestamp.Format("15:04:05"), entry.Message)
		s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color(color)).Render(line))
		s.WriteString("\n")
	}

	s.WriteString("\n")
	s.WriteString(lipgloss.NewStyle().Foreground(lipgloss.Color("241")).Render("Press 'q' to quit"))

	return baseStyle.Width(m.width - 2).Height(m.height - 2).Render(s.String())
}

func StartTUI(apiAddress string) error {
	p := tea.NewProgram(initialModel(apiAddress), tea.WithAltScreen())
	_, err := p.Run()
	return err
}
