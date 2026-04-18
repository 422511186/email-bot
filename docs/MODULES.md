# 模块详解

## 目录

- [main.go](#main-go)
- [core/bot.go](#corebot-go)
- [core/fetcher.go](#corefetcher-go)
- [core/forwarder.go](#coreforwarder-go)
- [core/state.go](#corestate-go)
- [config/config.go](#configconfig-go)
- [tui/app.go](#tuiapp-go)

---

## main.go

**职责**：程序入口，命令行参数解析，启动各组件。

### 命令行参数

| 参数 | 类型 | 默认值 | 说明 |
|------|------|--------|------|
| `-config` | string | `config.yaml` | 配置文件路径 |
| `-version` | bool | false | 显示版本并退出 |

### 代码流程

```go
func main() {
    // 1. 解析命令行参数
    flag.Parse()

    // 2. 显示版本
    if *showVersion {
        fmt.Println("email-bot 版本:", Version)
        return
    }

    // 3. 加载配置
    cfg, err := config.Load(*configPath)

    // 4. 创建机器人引擎
    bot, err := core.NewBot(cfg)

    // 5. 在 goroutine 中启动机器人
    go bot.Run()

    // 6. 启动 TUI
    model := tui.NewModel(bot, cfg)
    p := tea.NewProgram(model,
        tea.WithAltScreen(),       // 全屏模式
        tea.WithMouseCellMotion(), // 鼠标支持
    )
    p.Run()

    // 7. 优雅退出
    bot.Stop()
}
```

---

## core/bot.go

**职责**：机器人核心引擎，管理轮询调度、事件发布、并发处理。

### 核心结构

```go
type Bot struct {
    cfg      *config.Config    // 配置引用
    state    *State            // 状态管理器
    events   chan Event        // 事件通道（TUI 订阅）
    stopCh   chan struct{}     // 停止信号
    pollNow  chan struct{}     // 立即轮询信号
    mu       sync.RWMutex      // 状态保护
    statuses map[string]*MailboxStatus  // 邮箱状态缓存
    nextPoll time.Time         // 下次轮询时间
}
```

### 关键函数

#### `NewBot(cfg *config.Config) (*Bot, error)`

创建 Bot 实例，初始化状态和状态映射。

#### `Run()`

主循环，在 goroutine 中运行：
- 执行初始轮询
- 定时器触发轮询
- 监听手动触发信号
- 监听停止信号

#### `runPollCycle()`

轮询周期处理：
1. 发布"开始轮询"事件
2. 并发轮询所有源邮箱（WaitGroup）
3. 保存状态到磁盘
4. 发布"轮询完成"事件

#### `pollSource(src config.SourceAccount)`

处理单个源邮箱：
1. 标记为轮询中
2. 获取上次 UID
3. 调用 Fetcher 获取新邮件
4. 逐封转发（带延迟）
5. 更新状态

#### `TriggerPoll()`

请求立即轮询，非阻塞发送信号。

---

## core/fetcher.go

**职责**：IMAP 协议封装，邮件增量获取。

### 核心结构

```go
type FetchedEmail struct {
    UID     uint32    // IMAP UID
    Subject string    // 邮件主题
    From    string    // 发件人
    Date    time.Time // 日期
    Raw     []byte    // 完整 RFC-822 数据
}

type FetchResult struct {
    Emails     []FetchedEmail  // 获取的邮件
    NewLastUID uint32           // 新的最大 UID
}
```

### 关键函数

#### `FetchNewEmails(src config.SourceAccount, lastUID uint32, initialized bool) (FetchResult, error)`

核心函数，实现增量获取：

```go
// 首次运行
if !initialized {
    maxUID := findMaxUID()  // 获取当前最大 UID
    return FetchResult{NewLastUID: maxUID}  // 不返回邮件
}

// 正常运行
newUIDs := searchUIDs(lastUID)  // 搜索 > lastUID 的 UID
emails := fetchMessages(newUIDs)  // 下载邮件
return FetchResult{Emails: emails, NewLastUID: maxUID}
```

#### `findMaxUID(c *client.Client) (uint32, error)`

搜索所有 UID，返回最大值：
```go
set.AddRange(1, 0)  // 1:* 所有消息
criteria.Uid = set
uids, _ := c.UidSearch(criteria)
return uids[len(uids)-1]  // 升序排列，最后是最大
```

#### `fetchMessages(c *client.Client, uids []uint32, ...)`

批量获取邮件：
- 使用 UidFetch 批量获取
- 提取 Envelope 信息（主题、发件人、日期）
- 获取 RFC-822 完整数据

---

## core/forwarder.go

**职责**：SMTP 协议封装，网络超时控制，MIME 编码解析及连接复用的邮件转发。

### 关键结构与函数

#### `SMTPForwarder`

维护长生命周期的 SMTP Client 和信封发件人信息：

```go
type SMTPForwarder struct {
    client *smtp.Client
    from   string
}
```

#### `NewSMTPForwarder(cfg config.SMTPConfig) (*SMTPForwarder, error)`

拨号建立 SMTP 连接：
- 内置 30 秒的 `net.Dialer` 网络拨号超时控制
- 根据端口 (465/587) 自动处理隐式 TLS 与 STARTTLS
- 完成身份认证

#### `(f *SMTPForwarder) ForwardEmail(email FetchedEmail, targets []string) error`

复用现有 SMTP 连接转发邮件：
- 发送 MAIL FROM 和 RCPT TO 指令
- 将处理后的原邮件数据（`prependResentHeaders`）写入数据流中

#### `modifySubject(original []byte, prefix string) []byte`

安全修改邮件主题（Subject）：
- 定位并切分出 Headers 和 Body
- 支持多行折叠 (Folding) 的 Header 合并
- 使用 `mime.WordDecoder` 智能解码原有的 Base64/Quoted-Printable 主题
- 将自定义前缀与原主题拼接后，再通过 `mime.BEncoding` 安全地重新编码
- 避免破坏原邮件结构导致乱码

---

## core/state.go

**职责**：状态持久化，UID 高水位管理。

### 核心结构

```go
type sourceState struct {
    LastUID     uint32  // 最后处理的 UID
    Initialized bool    // 是否已完成首次运行
}

type State struct {
    mu      sync.RWMutex
    Sources map[string]*sourceState  // 按用户名索引
}
```

### 关键函数

#### `LoadState(path string) (*State, error)`

从文件加载状态，文件不存在时返回空状态。

#### `Save(path string) error`

原子写入状态：
1. 写入临时文件（600 权限）
2. 重命名覆盖原文件

#### `GetLastUID(username string) uint32`

获取用户最后处理的 UID。

#### `SetLastUID(username string, uid uint32)`

更新 UID 并标记为已初始化。

---

## config/config.go

**职责**：配置加载、默认值应用、配置验证。

### 核心结构

```go
type SourceAccount struct {
    Name     string   // 显示名称
    Host     string   // IMAP 主机
    Port     int      // IMAP 端口（默认 993）
    Username string   // 完整邮箱地址
    Password string   // 应用密码/授权码
    Mailbox  string   // 邮箱文件夹（默认 INBOX）
    Targets  []string // 目标地址列表
}

type SMTPConfig struct {
    Host     string  // SMTP 主机
    Port     int     // SMTP 端口（默认 587）
    Username string  // 用户名
    Password string  // 密码
    From     string  // 发件人地址
}

type Config struct {
    PollInterval int              // 轮询间隔（秒，默认 60）
    ForwardDelay int             // 转发间隔（毫秒，默认 1000）
    Sources      []SourceAccount // 源邮箱列表
    SMTP         SMTPConfig      // SMTP 配置
    StateFile    string          // 状态文件路径
}
```

### 默认值

| 字段 | 默认值 |
|------|--------|
| Sources[*].Port | 993 |
| Sources[*].Mailbox | INBOX |
| Sources[*].Name | Username |
| SMTP.Port | 587 |
| SMTP.From | SMTP.Username |
| PollInterval | 60 |
| ForwardDelay | 1000 |
| StateFile | ~/.email-bot/state.json |

---

## tui/app.go

**职责**：Bubbletea TUI 实现，双面板布局。

### 布局结构

```
┌─────────────────────────────────────────────────────────────────┐
│  📧 Email Bot        ● 运行中  |  下次轮询 47s  |  10:32:15    │
│  ──────────────────────────────────────────────────────────────  │
│  ┌─ 邮箱列表 ─────────────┐  ┌─ 活动日志 ─────────────────────┐ │
│  │ ✓ 工作邮箱 Gmail      │  │ 10:31:28 邮件机器人已启动    │ │
│  │   上次: 10:31:28      │  │ 10:31:29 正在轮询...         │ │
│  │   已转发: 3           │  │ 10:31:31 转发成功            │ │
│  │ ○ QQ 邮箱            │  │                               │ │
│  └──────────────────────┘  └───────────────────────────────┘ │
│  ──────────────────────────────────────────────────────────────  │
│   q 退出  r 立即轮询  ↑↓ 选择邮箱  PgUp/Dn 滚动日志           │
└─────────────────────────────────────────────────────────────────┘
```

### 样式定义

```go
// 颜色定义
cPrimary   = "#7C3AED"  // 紫色（标题）
cSuccess   = "#10B981"  // 绿色（成功）
cError     = "#F87171"  // 红色（错误）
cWarning   = "#FBBF24"  // 黄色（警告）
cMuted     = "#6B7280"  // 灰色（次要）
```

### 快捷键

| 按键 | 功能 |
|------|------|
| `q` / `Ctrl+C` | 退出程序 |
| `r` | 立即触发轮询 |
| `↑` / `k` | 选择上一个邮箱 |
| `↓` / `j` | 选择下一个邮箱 |
| `Tab` | 切换焦点（邮箱列表 ↔ 日志） |
| `PgUp` / `u` | 日志向上滚动 |
| `PgDn` / `d` | 日志向下滚动 |
| `g` | 跳转到日志顶部 |
| `G` | 跳转到日志底部 |

### 状态图标

| 图标 | 状态 |
|------|------|
| `⟳` (黄色) | 正在轮询 |
| `✗` (红色) | 上次轮询失败 |
| `✓` (绿色) | 正常（已同步） |
| `○` (灰色) | 等待首次轮询 |

### 日志颜色

根据日志前缀自动着色：

- `✅` `✓` `✉️` `🔖` → 绿色（成功）
- `❌` → 红色（错误）
- `⚠️` → 黄色（警告）
- `📬` `🔄` `📨` → 紫色（信息）
- `💤` → 灰色（静默）
