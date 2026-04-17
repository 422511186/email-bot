package tui

import (
	"fmt"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"email-bot/config"
	"email-bot/core"
)

// ── 调色板 ──────────────────────────────────────────────────────────────

var (
	cPrimary   = lipgloss.Color("#7C3AED")
	cSuccess   = lipgloss.Color("#10B981")
	cError     = lipgloss.Color("#F87171")
	cWarning   = lipgloss.Color("#FBBF24")
	cMuted     = lipgloss.Color("#6B7280")
	cText      = lipgloss.Color("#F9FAFB")
	cTextDim   = lipgloss.Color("#9CA3AF")
	cBorder    = lipgloss.Color("#374151")
	cActiveBdr = lipgloss.Color("#7C3AED")
	cSelected  = lipgloss.Color("#4C1D95")
)

// ── 样式 ─────────────────────────────────────────────────────────────────

var (
	styleTitle = lipgloss.NewStyle().
			Foreground(cPrimary).
			Bold(true)

	styleDivider = lipgloss.NewStyle().
			Foreground(cBorder)

	stylePanel = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(cBorder)

	styleActivePanel = lipgloss.NewStyle().
				Border(lipgloss.RoundedBorder()).
				BorderForeground(cActiveBdr)

	stylePanelTitle = lipgloss.NewStyle().
			Foreground(cPrimary).
			Bold(true)

	styleMailboxName = lipgloss.NewStyle().
				Foreground(cText).
				Bold(true)

	styleSelected = lipgloss.NewStyle().
			Background(cSelected).
			Foreground(cText).
			Bold(true)

	styleMeta = lipgloss.NewStyle().
			Foreground(cTextDim)

	styleSuccess = lipgloss.NewStyle().Foreground(cSuccess)
	styleError   = lipgloss.NewStyle().Foreground(cError)
	styleWarning = lipgloss.NewStyle().Foreground(cWarning)
	stylePrimary = lipgloss.NewStyle().Foreground(cPrimary)
	styleMuted   = lipgloss.NewStyle().Foreground(cMuted)

	styleKey     = lipgloss.NewStyle().Foreground(lipgloss.Color("#A78BFA")).Bold(true)
	styleKeyHint = lipgloss.NewStyle().Foreground(cMuted)
	styleStatus  = lipgloss.NewStyle().Foreground(cTextDim)
)

// ── 消息类型（bubbletea）────────────────────────────────────────────────

type tickMsg time.Time
type botEventMsg core.Event

// ── 日志条目 ─────────────────────────────────────────────────────────────

type logLine struct {
	ts  time.Time
	msg string
}

// ── 模型 ─────────────────────────────────────────────────────────────────

const (
	leftPanelW  = 38
	maxLogLines = 500
	headerLines = 3
	footerLines = 3
)

// Model 是根 bubbletea 模型。
type Model struct {
	bot *core.Bot
	cfg *config.Config

	statuses []*core.MailboxStatus
	logs     []logLine

	selected  int  // 左侧面板中选中的邮箱索引
	logOffset int  // 右侧面板中的滚动偏移
	logFocus  bool // true 表示右侧面板获得焦点

	width  int
	height int
}

// NewModel 构造初始 TUI 模型。
func NewModel(bot *core.Bot, cfg *config.Config) Model {
	return Model{
		bot:      bot,
		cfg:      cfg,
		statuses: bot.GetStatuses(),
	}
}

func (m Model) Init() tea.Cmd {
	return tea.Batch(
		listenForEvent(m.bot.Events()),
		tick(),
	)
}

// ── 更新 ─────────────────────────────────────────────────────────────────

func (m Model) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {

	case tea.WindowSizeMsg:
		m.width = msg.Width
		m.height = msg.Height

	case tickMsg:
		return m, tick()

	case botEventMsg:
		e := core.Event(msg)
		switch e.Kind {
		case core.EventLog:
			m.logs = append(m.logs, logLine{ts: e.Timestamp, msg: e.Message})
			if len(m.logs) > maxLogLines {
				m.logs = m.logs[len(m.logs)-maxLogLines:]
			}
			// 如果用户没有手动向上滚动，则自动滚动到底部。
			vh := m.logViewHeight()
			maxOff := lenMax0(len(m.logs)-vh)
			if m.logOffset >= maxOff-2 || m.logOffset == 0 {
				m.logOffset = maxOff
			}
		case core.EventMailboxUpdate:
			if e.Status != nil {
				m.upsertStatus(e.Status)
			}
		}
		return m, listenForEvent(m.bot.Events())

	case tea.KeyMsg:
		return m.handleKey(msg)
	}

	return m, nil
}

func (m Model) handleKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch msg.String() {
	case "q", "Q", "ctrl+c":
		return m, tea.Quit

	case "r", "R":
		m.bot.TriggerPoll()
		m.logs = append(m.logs, logLine{
			ts:  time.Now(),
			msg: "🖱️  用户触发了手动轮询",
		})

	case "tab":
		m.logFocus = !m.logFocus

	// 左侧面板导航（邮箱列表）
	case "up", "k":
		if !m.logFocus && m.selected > 0 {
			m.selected--
		}
	case "down", "j":
		if !m.logFocus && m.selected < len(m.statuses)-1 {
			m.selected++
		}

	// 右侧面板日志滚动
	case "pgup", "u":
		m.logOffset = imax(0, m.logOffset-m.logViewHeight())
	case "pgdn", "d":
		m.logOffset = imin(lenMax0(len(m.logs)-m.logViewHeight()), m.logOffset+m.logViewHeight())
	case "g":
		m.logOffset = 0
	case "G":
		m.logOffset = lenMax0(len(m.logs) - m.logViewHeight())
	}

	return m, nil
}

// ── 视图 ─────────────────────────────────────────────────────────────────

func (m Model) View() string {
	if m.width == 0 {
		return "加载中…"
	}
	return strings.Join([]string{
		m.renderHeader(),
		m.renderBody(),
		m.renderFooter(),
	}, "\n")
}

// renderHeader 渲染带有标题和下次轮询倒计时的顶部栏。
func (m Model) renderHeader() string {
	title := styleTitle.Render("📧 Email Bot")

	nextIn := time.Until(m.bot.NextPoll())
	if nextIn < 0 {
		nextIn = 0
	}
	rightText := styleSuccess.Render(
		fmt.Sprintf("● 运行中  |  下次轮询 %ds  |  %s",
			int(nextIn.Seconds()),
			time.Now().Format("15:04:05"),
		),
	)

	gap := m.width - visWidth(title) - visWidth(rightText) - 2
	if gap < 0 {
		gap = 0
	}
	line1 := title + strings.Repeat(" ", gap) + rightText
	line2 := styleDivider.Render(strings.Repeat("─", m.width))
	return line1 + "\n" + line2
}

// renderBody 构建双面板内容区域。
func (m Model) renderBody() string {
	contentH := m.height - headerLines - footerLines
	if contentH < 4 {
		contentH = 4
	}

	innerH := contentH - 2
	innerLeftW := leftPanelW - 2
	innerRightW := m.width - leftPanelW - 3 - 2

	left := m.renderMailboxPanel(innerLeftW, innerH)
	right := m.renderLogPanel(innerRightW, innerH)

	leftPanel := stylePanel
	rightPanel := stylePanel
	if !m.logFocus {
		leftPanel = styleActivePanel
	} else {
		rightPanel = styleActivePanel
	}

	leftBox := leftPanel.Width(leftPanelW).Height(contentH).Render(left)
	rightBox := rightPanel.Width(m.width - leftPanelW - 3).Height(contentH).Render(right)

	return lipgloss.JoinHorizontal(lipgloss.Top, leftBox, " ", rightBox)
}

// renderMailboxPanel 渲染列出所有源账户的左侧面板。
func (m Model) renderMailboxPanel(w, h int) string {
	lines := []string{
		stylePanelTitle.Render("邮箱列表"),
		"",
	}

	for i, s := range m.statuses {
		// 状态图标
		var icon, iconLine string
		switch {
		case s.IsPolling:
			icon = styleWarning.Render("⟳")
		case s.LastError != nil:
			icon = styleError.Render("✗")
		case !s.LastPoll.IsZero():
			icon = styleSuccess.Render("✓")
		default:
			icon = styleMuted.Render("○")
		}

		name := clipRune(s.AccountName, w-3)
		var nameRendered string
		if i == m.selected {
			nameRendered = styleSelected.Width(w - 2).Render(name)
		} else {
			nameRendered = styleMailboxName.Render(name)
		}

		iconLine = icon + " " + nameRendered
		lines = append(lines, iconLine)

		// 错误信息或上次轮询时间
		if s.LastError != nil {
			errStr := clipRune(s.LastError.Error(), w-3)
			lines = append(lines, "  "+styleError.Render(errStr))
		} else if !s.LastPoll.IsZero() {
			lines = append(lines, "  "+styleMeta.Render(fmt.Sprintf(
				"上次: %s   已转发: %d",
				s.LastPoll.Format("15:04:05"),
				s.TotalFwded,
			)))
		} else {
			lines = append(lines, "  "+styleMeta.Render("等待首次轮询…"))
		}

		// 选中此邮箱时显示目标地址
		if i == m.selected {
			for _, src := range m.cfg.Sources {
				if src.Username == s.Username {
					lines = append(lines, "  "+styleMuted.Render(
						fmt.Sprintf("→ 目标数: %d 个:", len(src.Targets)),
					))
					for _, t := range src.Targets {
						lines = append(lines, "    "+stylePrimary.Render(clipRune(t, w-5)))
					}
				}
			}
		}

		lines = append(lines, "")
	}

	// 填充至高度
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	return strings.Join(lines, "\n")
}

// renderLogPanel 渲染带有可滚动活动日志的右侧面板。
func (m Model) renderLogPanel(w, h int) string {
	vh := h - 2

	lines := []string{
		stylePanelTitle.Render("活动日志"),
		"",
	}

	// 限制滚动偏移
	maxOff := lenMax0(len(m.logs) - vh)
	off := imin(m.logOffset, maxOff)
	if off < 0 {
		off = 0
	}

	end := off + vh
	if end > len(m.logs) {
		end = len(m.logs)
	}
	visible := m.logs[off:end]

	if len(m.logs) == 0 {
		lines = append(lines, styleMeta.Render("  等待事件…"))
	}

	for _, entry := range visible {
		ts := styleMuted.Render(entry.ts.Format("15:04:05"))
		msg := coloriseLog(entry.msg, w-10)
		lines = append(lines, ts+" "+msg)
	}

	// 滚动指示器
	if len(m.logs) > vh {
		total := len(m.logs)
		pct := 0
		if maxOff > 0 {
			pct = (off * 100) / maxOff
		}
		indicator := styleMeta.Render(fmt.Sprintf(" [%d/%d  %d%%]", off+len(visible), total, pct))
		scroll := styleMuted.Render(strings.Repeat("─", imax(0, w-visWidth(indicator)))) + indicator
		lines = append(lines, scroll)
	}

	// 填充至高度
	for len(lines) < h {
		lines = append(lines, "")
	}
	if len(lines) > h {
		lines = lines[:h]
	}

	return strings.Join(lines, "\n")
}

// renderFooter 渲染底部按键提示。
func (m Model) renderFooter() string {
	divider := styleDivider.Render(strings.Repeat("─", m.width))

	hint := func(k, desc string) string {
		return styleKey.Render(k) + styleKeyHint.Render(" "+desc)
	}

	var focusHint string
	if m.logFocus {
		focusHint = hint("Tab", "→ 邮箱")
	} else {
		focusHint = hint("Tab", "→ 日志")
	}

	hints := strings.Join([]string{
		hint("q", "退出"),
		hint("r", "立即轮询"),
		hint("↑↓/j/k", "选择邮箱"),
		hint("PgUp/PgDn", "滚动日志"),
		hint("g/G", "日志顶部/底部"),
		focusHint,
	}, "  ")

	// 如果太宽则截断
	if visWidth(hints) > m.width-2 {
		hints = hint("q", "退出") + "  " + hint("r", "立即轮询") + "  " + hint("↑↓", "选择") + "  " + hint("PgUp/Dn", "滚动")
	}

	return divider + "\n" + styleStatus.Render(" ") + hints
}

// ── 辅助函数 ─────────────────────────────────────────────────────────────

// logViewHeight 返回可用的日志行数。
func (m Model) logViewHeight() int {
	h := m.height - headerLines - footerLines - 4
	if h < 1 {
		return 1
	}
	return h
}

// upsertStatus 更新匹配的状态或追加新状态。
func (m *Model) upsertStatus(s *core.MailboxStatus) {
	for i, st := range m.statuses {
		if st.Username == s.Username {
			m.statuses[i] = s
			return
		}
	}
	m.statuses = append(m.statuses, s)
}

// coloriseLog 根据日志行的前缀 emoji/文字选择颜色。
func coloriseLog(msg string, maxW int) string {
	msg = clipRune(msg, maxW)
	switch {
	case strings.HasPrefix(msg, "✉️"),
		strings.HasPrefix(msg, "✅"),
		strings.HasPrefix(msg, "✓"),
		strings.HasPrefix(msg, "🔖"):
		return styleSuccess.Render(msg)
	case strings.HasPrefix(msg, "❌"):
		return styleError.Render(msg)
	case strings.HasPrefix(msg, "⚠️"):
		return styleWarning.Render(msg)
	case strings.HasPrefix(msg, "📬"),
		strings.HasPrefix(msg, "🔄"),
		strings.HasPrefix(msg, "📨"):
		return stylePrimary.Render(msg)
	case strings.HasPrefix(msg, "💤"):
		return styleMeta.Render(msg)
	default:
		return lipgloss.NewStyle().Foreground(cText).Render(msg)
	}
}

// listenForEvent 返回一个阻塞至下一个机器人事件的 Cmd。
func listenForEvent(events <-chan core.Event) tea.Cmd {
	return func() tea.Msg {
		return botEventMsg(<-events)
	}
}

// tick 返回每秒 tick 命令。
func tick() tea.Cmd {
	return tea.Tick(time.Second, func(t time.Time) tea.Msg {
		return tickMsg(t)
	})
}

// visWidth 返回字符串的显示宽度（去除 ANSI 转义码）。
func visWidth(s string) int {
	return lipgloss.Width(s)
}

// clipRune 将字符串截断至最多 n 个字符。
func clipRune(s string, n int) string {
	if n <= 0 {
		return ""
	}
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	if n <= 1 {
		return string(r[:n])
	}
	return string(r[:n-1]) + "…"
}

func imax(a, b int) int {
	if a > b {
		return a
	}
	return b
}

func imin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func lenMax0(n int) int {
	if n < 0 {
		return 0
	}
	return n
}
