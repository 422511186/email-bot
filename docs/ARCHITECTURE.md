# 架构设计

## 整体架构

Email Bot 采用**事件驱动架构**，核心组件通过 Channel 进行通信。

```
┌─────────────────────────────────────────────────────────────────┐
│                         Email Bot                                 │
│                                                                  │
│  ┌──────────────┐         ┌─────────────────┐                   │
│  │   Config     │────────▶│      Bot         │                   │
│  │   Loader     │         │   (调度器)       │                   │
│  └──────────────┘         └────────┬────────┘                   │
│                                    │                             │
│                          ┌─────────▼─────────┐                   │
│                          │   Event Channel   │◄────── TUI 订阅    │
│                          └─────────┬─────────┘                   │
│                                    │                             │
│         ┌──────────────────────────┼──────────────────────────┐   │
│         │                          │                          │   │
│  ┌──────▼──────┐          ┌──────▼──────┐          ┌──────▼──────┐
│  │   Fetcher   │          │  Forwarder   │          │    State    │
│  │   (获取)    │─────────▶│   (转发)      │          │   (持久化)   │
│  └─────────────┘          └──────────────┘          └─────────────┘
│         │                          │                          │
└─────────┼──────────────────────────┼──────────────────────────┘
          │                          │
          ▼                          ▼
    ┌──────────┐              ┌──────────┐
    │  IMAP    │              │   SMTP   │
    │  Server  │              │  Server  │
    └──────────┘              └──────────┘
```

## 模块职责

### 1. 配置层 (config/)

```
config/config.go
├── Load()           # 加载并验证 YAML 配置
├── validate()        # 配置校验
└── Config 结构体    # 配置数据模型
```

**职责**：
- 读取 `config.yaml` 配置文件
- 应用默认值（端口、超时等）
- 验证必填字段
- 返回标准化的配置结构

### 2. 核心引擎层 (core/)

#### Bot (`bot.go`)

```
Bot
├── Run()             # 主循环，定时触发轮询
├── pollSource()      # 处理单个源邮箱
└── Event Channel     # 事件发布
```

**职责**：
- 管理轮询调度（定时 + 手动触发）
- 并发处理多个源邮箱
- 发布事件给 TUI
- 错误处理与日志记录

#### Fetcher (`fetcher.go`)

```
Fetcher
├── FetchNewEmails()  # 增量获取新邮件
├── findMaxUID()      # 查找最大 UID（首次运行）
├── searchUIDs()      # 搜索新邮件 UID
└── fetchMessages()   # 下载邮件内容
```

**职责**：
- IMAP 连接管理
- UID 追踪（增量获取关键）
- 邮件内容下载（完整 RFC-822）

#### Forwarder (`forwarder.go`)

```
Forwarder
├── ForwardEmail()    # 发送邮件
├── prependResentHeaders()  # 添加 Resent-* 头
└── sendMailImplicitTLS()   # 隐式 TLS 发送
```

**职责**：
- SMTP 连接管理
- 邮件转发（保留原始内容）
- 支持 STARTTLS 和 SSL

#### State (`state.go`)

```
State
├── LoadState()       # 从文件加载状态
├── Save()            # 原子写入状态
├── GetLastUID()      # 获取最后 UID
└── SetLastUID()      # 更新 UID
```

**职责**：
- UID 高水位持久化
- 首次运行标记
- 原子文件写入

### 3. 用户界面层 (tui/)

```
TUI (Bubbletea)
├── Model              # 状态模型
├── Init()             # 初始化
├── Update()           # 消息处理
├── View()             # 渲染界面
└── 渲染函数
    ├── renderHeader()     # 头部：状态 + 倒计时
    ├── renderBody()       # 主体：双面板
    ├── renderMailboxPanel()  # 左面板：邮箱列表
    ├── renderLogPanel()      # 右面板：活动日志
    └── renderFooter()       # 底部：快捷键提示
```

**职责**：
- 双面板布局（邮箱列表 + 日志）
- 实时状态显示
- 用户交互处理
- 滚动和导航

## 事件流

### 轮询周期

```
Bot.Run()
  │
  ▼
定时器触发 / 手动触发
  │
  ▼
runPollCycle()
  │
  ├─────────────────────────────────────┐
  │  并发执行（goroutine）              │
  │                                     │
  ▼                                     ▼
pollSource(src1)              pollSource(srcN)
  │                                     │
  ▼                                     ▼
FetchNewEmails()              FetchNewEmails()
  │                                     │
  ├─────────────────────────────────────┤
  │                                     │
  ▼                                     ▼
ForwardEmail() ──────────▶ SMTP Server
  │                                     │
  └─────────────────────────────────────┘
                    │
                    ▼
              Bot.events Channel
                    │
                    ▼
              TUI.Update()
```

### 首次运行流程

```
首次启动
    │
    ▼
FetchNewEmails(initialized=false)
    │
    ▼
findMaxUID()  ──▶ 获取当前最大 UID
    │
    ▼
返回 newLastUID（不返回任何邮件）
    │
    ▼
SetLastUID(username, maxUID)
    │
    ▼
标记 Initialized = true
    │
    ▼
下次运行开始正常转发
```

## 数据模型

### UID 高水位机制

```
邮箱中的邮件 UID 是递增的：
UID: 1, 2, 3, 4, 5, 6, 7, ...

首次运行：
  → 记录 maxUID = 7
  → 不转发任何邮件

新邮件到达（UID = 8, 9）：
  → 查找 UID > 7 的邮件
  → 获取 UID 8, 9
  → 转发
  → 更新 maxUID = 9
```

### 状态文件结构

```json
{
  "sources": {
    "user1@gmail.com": {
      "last_uid": 9,
      "initialized": true
    },
    "user2@qq.com": {
      "last_uid": 15,
      "initialized": true
    }
  }
}
```

## 并发设计

### 多源邮箱并发

```go
var wg sync.WaitGroup
for _, src := range cfg.Sources {
    wg.Add(1)
    go func(src config.SourceAccount) {
        defer wg.Done()
        b.pollSource(src)  // 每个源邮箱独立 goroutine
    }(src)
}
wg.Wait()  // 等待所有源邮箱处理完成
```

### 状态保护

```go
// Bot 内部状态使用 RWMutex
type Bot struct {
    mu       sync.RWMutex
    statuses map[string]*MailboxStatus
}

// 读操作：共享锁
func (b *Bot) GetStatuses() []*MailboxStatus {
    b.mu.RLock()
    defer b.mu.RUnlock()
    // ...
}

// 写操作：独占锁
func (b *Bot) pollSource(...) {
    b.mu.Lock()
    status.IsPolling = true
    b.mu.Unlock()
    // ...
}
```

## 配置加载流程

```
main.go
    │
    ▼
config.Load("config.yaml")
    │
    ├── 读取 YAML 文件
    │
    ├── 解析到 Config 结构体
    │
    ├── 应用默认值
    │   ├── Port = 0 → 993
    │   ├── Mailbox = "" → "INBOX"
    │   └── ForwardDelay = 0 → 1000
    │
    └── 验证配置
        ├── 检查 Sources 非空
        ├── 检查 SMTP.Host 必填
        └── 检查 Sources[*].Targets 非空
            │
            ▼
        Bot(cfg)
```

## 错误处理策略

| 场景 | 处理方式 |
|------|----------|
| IMAP 连接失败 | 记录错误、标记状态、继续其他源 |
| 邮件获取失败 | 返回已获取的邮件 + 错误 |
| SMTP 发送失败 | 单封失败不影响其他邮件 |
| 状态保存失败 | 仅记录警告，不阻塞流程 |
| 配置无效 | 程序启动失败，输出错误信息 |

## 扩展点

如需扩展功能，可考虑：

1. **附件过滤**：在 `fetcher.go` 中解析 MIME
2. **邮件过滤**：在 `bot.go` 中添加过滤规则
3. **重试机制**：在 `forwarder.go` 中添加失败重试
4. **监控指标**：添加 Prometheus 指标暴露
5. **Webhook**：支持 HTTP 回调通知
