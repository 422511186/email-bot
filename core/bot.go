package core

import (
	"fmt"
	"strings"
	"sync"
	"time"

	"email-bot/config"
)

// ── 事件类型 ──────────────────────────────────────────────────────────────

// EventKind 对机器人事件进行分类。
type EventKind int

const (
	EventLog           EventKind = iota // 普通日志行
	EventMailboxUpdate                  // MailboxStatus 已变更
)

// Event 在 Bot.Events() 返回的通道上发送。
type Event struct {
	Kind      EventKind
	Message   string
	Timestamp time.Time
	Status    *MailboxStatus // 当 Kind == EventMailboxUpdate 时非空
}

// ── 邮箱状态 ──────────────────────────────────────────────────────────────

// MailboxStatus 是一个源账户当前状态的快照。
type MailboxStatus struct {
	AccountName string
	Username    string
	LastPoll    time.Time
	LastError   error
	TotalFwded  int
	IsPolling   bool
}

// ── 机器人 ───────────────────────────────────────────────────────────────

// Bot 是邮件转发引擎。
type Bot struct {
	cfg      *config.Config
	state    *State
	events   chan Event
	stopCh   chan struct{}
	pollNow  chan struct{}
	wg       sync.WaitGroup
	mu       sync.RWMutex
	statuses map[string]*MailboxStatus // 以 Username 为键
	nextPoll time.Time
}

// NewBot 创建 Bot 实例。调用者必须在 goroutine 中调用 Run()。
func NewBot(cfg *config.Config) (*Bot, error) {
	state, err := LoadState(cfg.StateFile)
	if err != nil {
		state = NewState()
	}

	statuses := make(map[string]*MailboxStatus, len(cfg.Sources))
	for _, src := range cfg.Sources {
		statuses[src.Username] = &MailboxStatus{
			AccountName: src.Name,
			Username:    src.Username,
		}
	}

	return &Bot{
		cfg:      cfg,
		state:    state,
		events:   make(chan Event, 512),
		stopCh:   make(chan struct{}),
		pollNow:  make(chan struct{}, 1),
		statuses: statuses,
	}, nil
}

// Events 返回只读通道，Bot 在此通道上发送事件。
// TUI 从此通道读取数据以更新显示。
func (b *Bot) Events() <-chan Event {
	return b.events
}

// NextPoll 返回下次计划轮询的时间。
func (b *Bot) NextPoll() time.Time {
	b.mu.RLock()
	defer b.mu.RUnlock()
	return b.nextPoll
}

// GetStatuses 返回所有邮箱状态的快照，按配置顺序排列。
func (b *Bot) GetStatuses() []*MailboxStatus {
	b.mu.RLock()
	defer b.mu.RUnlock()
	out := make([]*MailboxStatus, 0, len(b.cfg.Sources))
	for _, src := range b.cfg.Sources {
		if s, ok := b.statuses[src.Username]; ok {
			cp := *s
			out = append(out, &cp)
		}
	}
	return out
}

// TriggerPoll 请求立即执行轮询（非阻塞）。
func (b *Bot) TriggerPoll() {
	select {
	case b.pollNow <- struct{}{}:
	default:
	}
}

// Stop 信号通知机器人正常关闭，并等待当前处理中的轮询完成。
func (b *Bot) Stop() {
	close(b.stopCh)
	b.wg.Wait()
}

// Run 是机器人的主循环。在 goroutine 中调用。
func (b *Bot) Run() {
	b.wg.Add(1)
	defer b.wg.Done()

	b.emit(EventLog, "🚀 邮件机器人已启动 — 执行初始轮询…")
	b.runPollCycle()

	interval := time.Duration(b.cfg.PollInterval) * time.Second
	for {
		b.mu.Lock()
		b.nextPoll = time.Now().Add(interval)
		b.mu.Unlock()

		select {
		case <-time.After(interval):
			b.runPollCycle()
		case <-b.pollNow:
			b.runPollCycle()
		case <-b.stopCh:
			b.emit(EventLog, "🛑 邮件机器人正在安全停止中...")
			return
		}
	}
}

// runPollCycle 并发轮询所有源邮箱，然后持久化状态。
func (b *Bot) runPollCycle() {
	b.emit(EventLog, "🔄 开始轮询周期…")

	// 限制同时并发轮询的邮箱数为 5，防止协程和连接数过多导致资源耗尽
	const maxConcurrentSources = 5
	sem := make(chan struct{}, maxConcurrentSources)

	var wg sync.WaitGroup
	for _, src := range b.cfg.Sources {
		wg.Add(1)
		go func(src config.SourceAccount) {
			defer wg.Done()
			
			sem <- struct{}{}        // 获取信号量
			defer func() { <-sem }() // 释放信号量
			
			b.pollSource(src)
		}(src)
	}
	wg.Wait()

	if err := b.state.Save(b.cfg.StateFile); err != nil {
		b.emit(EventLog, fmt.Sprintf("⚠️  状态保存失败: %v", err))
	}
	b.emit(EventLog, "✅ 轮询周期完成")
}

// pollSource 处理一个源账户：获取 → 转发 → 更新状态。
func (b *Bot) pollSource(src config.SourceAccount) {
	// 标记为正在轮询
	b.mu.Lock()
	status := b.statuses[src.Username]
	status.IsPolling = true
	b.mu.Unlock()
	b.emitStatus(src.Username)

	defer func() {
		b.mu.Lock()
		status.IsPolling = false
		status.LastPoll = time.Now()
		b.mu.Unlock()
		b.emitStatus(src.Username)
	}()

	b.emit(EventLog, fmt.Sprintf("📬 正在轮询 %s (%s)…", src.Name, src.Username))

	lastUID := b.state.GetLastUID(src.Username)
	initialized := b.state.IsInitialized(src.Username)

	result, err := FetchNewEmails(src, lastUID, initialized)

	if err != nil {
		b.mu.Lock()
		status.LastError = err
		b.mu.Unlock()
		b.emit(EventLog, fmt.Sprintf("❌ %s: 获取时发生部分错误 — %v", src.Name, err))
		
		// 注意：不要直接 return！
		// 因为 FetchNewEmails 在分批拉取时，如果遇到错误，
		// 依然会返回在错误发生之前成功拉取到的部分邮件 (result.Emails)。
		// 如果这里直接 return，那些成功拉取的邮件将被丢弃并在下一次轮询中重复下载。
	} else {
		b.mu.Lock()
		status.LastError = nil
		b.mu.Unlock()
	}

	// 首次运行初始化 — 暂无邮件需要转发
	if !initialized {
		// 持久化初始高水位 UID (无论是否为空邮箱，都必须标记为已初始化)
		b.state.SetLastUID(src.Username, result.NewLastUID)
		_ = b.state.Save(b.cfg.StateFile) // 立即落盘

		b.emit(EventLog, fmt.Sprintf(
			"🔖 %s: 已初始化（UID 高水位 = %d，此后新邮件将被转发）",
			src.Name, result.NewLastUID,
		))
		return
	}

	if len(result.Emails) == 0 {
		b.emit(EventLog, fmt.Sprintf("💤 %s: 无新邮件", src.Name))
		// 如果当前有遇到无法解析跳过的坏邮件（导致 Emails 为空但 maxUID 推进了）
		// 我们也应该推进 LastUID，防止无限重试这封坏邮件
		if result.NewLastUID > lastUID {
			b.state.SetLastUID(src.Username, result.NewLastUID)
			_ = b.state.Save(b.cfg.StateFile)
		}
		return
	}

	b.emit(EventLog, fmt.Sprintf("📨 %s: 发现 %d 封新邮件", src.Name, len(result.Emails)))

	smtpClient, err := NewSMTPForwarder(b.cfg.SMTP)
	if err != nil {
		b.emit(EventLog, fmt.Sprintf("❌ SMTP 连接失败: %v", err))
		return
	}
	defer smtpClient.Close()

	forwardFailed := false

	for i, email := range result.Emails {
		if err := smtpClient.ForwardEmail(email, src.Targets); err != nil {
			b.emit(EventLog, fmt.Sprintf(
				"❌ 转发失败 \"%s\": %v",
				clip(email.Subject, 45), err,
			))
			// 如果转发失败，停止后续邮件的转发和状态更新
			// 确保失败的邮件在下一次轮询时被重新拉取重试
			forwardFailed = true
			break
		} else {
			b.mu.Lock()
			status.TotalFwded++
			b.mu.Unlock()
			b.emit(EventLog, fmt.Sprintf(
				"✉️  \"%s\"  →  %s",
				clip(email.Subject, 38),
				strings.Join(src.Targets, ", "),
			))
			
			// 只有成功转发后，才更新并持久化当前邮件的 UID
			b.state.SetLastUID(src.Username, email.UID)
			_ = b.state.Save(b.cfg.StateFile)

			b.emitStatus(src.Username)
		}
		// 邮件之间添加延迟，避免目标邮箱被识别为垃圾邮件或触发限流
		if i < len(result.Emails)-1 && b.cfg.ForwardDelay > 0 {
			time.Sleep(time.Duration(b.cfg.ForwardDelay) * time.Millisecond)
		}
	}

	// 仅当本批次所有邮件都成功转发（未触发 break），
	// 且有因为 body 损坏而被跳过的坏邮件（导致 result.NewLastUID 大于最后一个成功邮件的 UID）时，
	// 我们才使用 result.NewLastUID 兜底推进，跨过死信，防止下一轮死循环。
	if !forwardFailed {
		currentSavedUID := b.state.GetLastUID(src.Username)
		if result.NewLastUID > currentSavedUID {
			b.state.SetLastUID(src.Username, result.NewLastUID)
			_ = b.state.Save(b.cfg.StateFile)
		}
	}
}

// ── 辅助函数 ─────────────────────────────────────────────────────────────

func (b *Bot) emit(kind EventKind, msg string) {
	select {
	case b.events <- Event{Kind: kind, Message: msg, Timestamp: time.Now()}:
	default:
	}
}

func (b *Bot) emitStatus(username string) {
	b.mu.RLock()
	s, ok := b.statuses[username]
	if !ok {
		b.mu.RUnlock()
		return
	}
	cp := *s
	b.mu.RUnlock()

	select {
	case b.events <- Event{Kind: EventMailboxUpdate, Timestamp: time.Now(), Status: &cp}:
	default:
	}
}

// clip 将字符串截断至最多 n 个字符，超出部分用 "…" 替代。
func clip(s string, n int) string {
	r := []rune(s)
	if len(r) <= n {
		return s
	}
	return string(r[:n-1]) + "…"
}
