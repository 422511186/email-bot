# API 参考

## 目录

- [核心类型](#核心类型)
- [事件系统](#事件系统)
- [配置结构](#配置结构)
- [导出函数](#导出函数)

---

## 核心类型

### core.FetchedEmail

获取的邮件数据结构。

```go
type FetchedEmail struct {
    UID     uint32    // IMAP 唯一标识符
    Subject string    // 邮件主题
    From    string    // 发件人地址，格式："名称 <邮箱>" 或 "邮箱"
    Date    time.Time // 邮件日期
    Raw     []byte    // 完整 RFC-822 原始数据（包含附件、内联图片）
}
```

**说明**：
- `Raw` 字段包含邮件的完整 MIME 数据，附件和内联图片以 base64 编码存储
- 转发时直接使用 `Raw` 发送，保证内容完整性

### core.FetchResult

邮件获取结果。

```go
type FetchResult struct {
    Emails     []FetchedEmail // 获取的邮件列表（可能为空）
    NewLastUID uint32         // 当前最大 UID，需持久化
}
```

### core.MailboxStatus

邮箱状态快照（用于 TUI 显示）。

```go
type MailboxStatus struct {
    AccountName string     // 配置中的显示名称
    Username    string     // 邮箱地址
    LastPoll    time.Time  // 上次轮询时间
    LastError   error      // 上次轮询错误（nil 表示正常）
    TotalFwded  int        // 累计转发邮件数
    IsPolling   bool       // 是否正在轮询
}
```

### core.EventKind

事件类型枚举。

```go
const (
    EventLog           EventKind = 0  // 普通日志事件
    EventMailboxUpdate EventKind = 1  // 邮箱状态更新事件
)
```

### core.Event

机器人事件结构。

```go
type Event struct {
    Kind      EventKind        // 事件类型
    Message   string           // 日志消息（EventLog 时）
    Timestamp time.Time        // 事件时间
    Status    *MailboxStatus  // 状态快照（EventMailboxUpdate 时）
}
```

---

## 事件系统

### EventLog 事件消息示例

| 消息 | 含义 |
|------|------|
| `🚀 邮件机器人已启动` | 程序启动 |
| `🔄 开始轮询周期…` | 轮询开始 |
| `📬 正在轮询 xxx` | 正在获取 |
| `📨 发现 N 封新邮件` | 获取到邮件 |
| `✉️ "主题" → 目标` | 转发成功 |
| `❌ 转发失败` | 转发失败 |
| `💤 无新邮件` | 无新邮件 |
| `🔖 已初始化` | 首次运行完成 |
| `⚠️ 状态保存失败` | 状态写入失败 |
| `🛑 邮件机器人已停止` | 程序退出 |

### 订阅事件

```go
// 获取事件通道（只读）
events := bot.Events()

// 监听循环
for event := range events {
    switch event.Kind {
    case core.EventLog:
        fmt.Println(event.Message)
    case core.EventMailboxUpdate:
        updateUI(event.Status)
    }
}
```

---

## 配置结构

### config.SourceAccount

源邮箱配置。

```go
type SourceAccount struct {
    Name     string   `yaml:"name"`      // 显示名称
    Host     string   `yaml:"host"`      // IMAP 主机（如 imap.gmail.com）
    Port     int      `yaml:"port"`      // IMAP 端口（默认 993）
    Username string   `yaml:"username"`  // 完整邮箱地址
    Password string   `yaml:"password"`  // 应用密码或授权码
    Mailbox  string   `yaml:"mailbox"`   // 邮箱文件夹（默认 INBOX）
    Targets  []string `yaml:"targets"`   // 目标邮箱地址列表
}
```

### config.SMTPConfig

SMTP 发送配置。

```go
type SMTPConfig struct {
    Host     string `yaml:"host"`      // SMTP 主机（如 smtp.gmail.com）
    Port     int    `yaml:"port"`      // 端口：587 (STARTTLS) 或 465 (SSL)
    Username string `yaml:"username"`  // 用户名
    Password string `yaml:"password"`  // 密码
    From     string `yaml:"from"`      // 发件人地址（信封 MAIL FROM）
}
```

### config.Config

根配置结构。

```go
type Config struct {
    PollInterval int              `yaml:"poll_interval"`  // 轮询间隔（秒，默认 60）
    ForwardDelay int              `yaml:"forward_delay"`  // 邮件间隔（毫秒，默认 1000）
    Sources      []SourceAccount  `yaml:"sources"`        // 源邮箱列表
    SMTP         SMTPConfig       `yaml:"smtp"`           // SMTP 配置
    StateFile    string           `yaml:"state_file"`     // 状态文件路径
}
```

---

## 导出函数

### core.NewBot

```go
func NewBot(cfg *config.Config) (*Bot, error)
```

创建机器人实例。

**参数**：
- `cfg`: 已验证的配置对象

**返回值**：
- 成功：`*Bot` 实例
- 失败：返回 `nil` 和错误

**示例**：

```go
bot, err := core.NewBot(cfg)
if err != nil {
    log.Fatal(err)
}
go bot.Run()
```

### core.Bot.Run

```go
func (b *Bot) Run()
```

启动机器人主循环。

**注意**：必须在 goroutine 中调用。

```go
go bot.Run()
```

### core.Bot.Stop

```go
func (b *Bot) Stop()
```

优雅停止机器人。

```go
bot.Stop()  // 在 TUI 退出后调用
```

### core.Bot.Events

```go
func (b *Bot) Events() <-chan Event
```

获取事件通道（只读）。

```go
events := bot.Events()
```

### core.Bot.TriggerPoll

```go
func (b *Bot) TriggerPoll()
```

请求立即轮询（非阻塞）。

```go
bot.TriggerPoll()
```

### core.Bot.GetStatuses

```go
func (b *Bot) GetStatuses() []*MailboxStatus
```

获取所有邮箱状态快照。

```go
statuses := bot.GetStatuses()
for _, s := range statuses {
    fmt.Printf("%s: %d 封已转发\n", s.AccountName, s.TotalFwded)
}
```

### core.Bot.NextPoll

```go
func (b *Bot) NextPoll() time.Time
```

获取下次计划轮询时间。

```go
next := bot.NextPoll()
fmt.Println("下次轮询:", next)
```

### core.FetchNewEmails

```go
func FetchNewEmails(src config.SourceAccount, lastUID uint32, initialized bool) (FetchResult, error)
```

增量获取新邮件。

**参数**：
- `src`: 源邮箱配置
- `lastUID`: 上次处理的 UID
- `initialized`: 是否已完成首次运行

**返回值**：
- `FetchResult`: 包含邮件列表和新的最大 UID

### core.ForwardEmail

```go
func ForwardEmail(smtpCfg config.SMTPConfig, email FetchedEmail, targets []string) error
```

转发单封邮件。

**参数**：
- `smtpCfg`: SMTP 配置
- `email`: 邮件数据
- `targets`: 目标地址列表

**返回值**：
- 成功：`nil`
- 失败：错误信息

### config.Load

```go
func Load(path string) (*Config, error)
```

加载并验证配置。

**参数**：
- `path`: 配置文件路径

**返回值**：
- 成功：`*Config` 和 `nil`
- 失败：`nil` 和错误信息

### core.LoadState

```go
func LoadState(path string) (*State, error)
```

从文件加载状态。

### core.State.Save

```go
func (s *State) Save(path string) error
```

原子写入状态文件。

---

## 依赖关系图

```
main.go
  │
  ├─► config.Load()
  │     └─► validate()
  │
  └─► core.NewBot()
        │
        ├─► LoadState()
        │
        └─► Bot.Run()
              │
              ├─► pollSource()
              │     ├─► FetchNewEmails()
              │     │     ├─► client.DialTLS()
              │     │     ├─► findMaxUID()
              │     │     ├─► searchUIDs()
              │     │     └─► fetchMessages()
              │     │
              │     └─► ForwardEmail()
              │           └─► sendMailImplicitTLS() / smtp.SendMail()
              │
              └─► state.Save()

tui.NewModel()
  │
  └─► bot.Events() ──► Event Channel ◄── bot.pollSource()
```
