# 📧 Email Bot

一个使用 Go 语言编写的终端邮件转发机器人。
轮询多个 IMAP 邮箱，增量获取新邮件，并将其转发到一个或多个目标地址 —— 全部通过实时 TUI 仪表板展示。

```
╭─ Email Bot ─────────────────────────── ● 运行中 | 下次轮询 47s | 10:32:15 ─╮
│ ╭─ 邮箱列表 ─────────────────────╮ ╭─ 活动日志 ──────────────────────────╮ │
│ │ 邮箱列表                        │ │ 活动日志                            │ │
│ │                                │ │                                     │ │
│ │ ✓ 工作邮箱 Gmail                │ │ 10:31:28 🚀 邮件机器人已启动         │ │
│ │   上次: 10:31:28  已转发: 3    │ │ 10:31:28 🔄 开始轮询周期…           │ │
│ │   → 目标数: 2 个:               │ │ 10:31:29 📬 正在轮询工作邮箱 Gmail…    │ │
│ │     archive@co.com              │ │ 10:31:31 ✉️  "Hello team" → arch@… │ │
│ │     notify@co.com               │ │ 10:31:31 ✅ 轮询周期完成              │ │
│ │                                │ │ 10:31:31 💤 QQ 邮箱: 无新邮件         │ │
│ │ ✓ QQ 邮箱                       │ │                                     │ │
│ │   上次: 10:31:30  已转发: 0    │ │                                     │ │
│ ╰─────────────────────────────────╯ ╰─────────────────────────────────────────╯ │
│──────────────────────────────────────────────────────────────────────────────────│
│  q 退出  r 立即轮询  ↑↓/j/k 选择邮箱  PgUp/PgDn 滚动日志  g/G 顶部/底部          │
╰──────────────────────────────────────────────────────────────────────────────────╯
```

---

## 功能特性

- **增量拉取** — 追踪每个邮箱的已见最大 IMAP UID；仅获取新邮件
- **首次运行保护** — 首次启动时记录当前邮箱状态，不转发历史邮件
- **一对多** — 一个源邮箱 → 多个目标地址
- **多对多** — 多个源邮箱，每个可配置独立的目标列表
- **SMTP 传输** — 支持 STARTTLS（587 端口）和隐式 TLS/SSL（465 端口）
- **持久化状态** — 重启后状态保留（`~/.email-bot/state.json`）
- **实时 TUI** — 双面板仪表板：左侧邮箱状态，右侧可滚动活动日志
- **跨平台** — Windows、Linux (amd64/arm64)、macOS (Intel + M系列) 单二进制

---

## 快速开始

### 1. 前置要求

```bash
# Go 1.21 或更高版本
go version
```

### 2. 克隆并安装依赖

```bash
git clone https://github.com/yourname/email-bot.git
cd email-bot
make deps
```

### 3. 配置

```bash
make init-config       # 将 config.yaml.example → config.yaml
$EDITOR config.yaml    # 填写你的 IMAP/SMTP 凭据
```

### 4. 运行

```bash
make run
# 或
go run . -config config.yaml
```

### 5. 构建发布版本

```bash
make build             # 当前平台
make build-all         # 所有平台 → dist/
```

---

## 配置说明

`config.yaml`（完整注释示例见 `config.yaml.example`）：

```yaml
poll_interval: 120   # 轮询间隔秒数

sources:
  # 一对多：一个收件箱 → 两个目标
  - name:     "工作邮箱 Gmail"
    host:     imap.gmail.com
    port:     993
    username: you@gmail.com
    password: "应用密码"
    targets:
      - archive@company.com
      - notifications@company.com

  # 多对多配置的一部分
  - name:     "QQ 邮箱"
    host:     imap.qq.com
    port:     993
    username: 12345@qq.com
    password: "QQ 授权码"
    targets:
      - personal@example.com

smtp:
  host:     smtp.gmail.com
  port:     587          # 587 = STARTTLS，465 = SSL
  username: sender@gmail.com
  password: "应用密码"
  from:     sender@gmail.com
```

### Gmail 配置

1. 在 Gmail 设置中启用 IMAP：查看所有设置 → 转发和 POP/IMAP
2. 创建应用密码：Google 账户 → 安全性 → 两步验证 → 应用密码
3. 使用 16 位应用密码作为 `password`

### QQ 邮箱配置

1. 登录 QQ 邮箱 → 设置 → 账户 → 开启 IMAP/SMTP 服务
2. 获取授权码，填写到 `password` 字段

---

## TUI 操作说明

| 按键 | 功能 |
|-----|------|
| `q` / `Ctrl+C` | 退出程序 |
| `r` | 立即触发轮询 |
| `↑` / `k` | 选择上一个邮箱 |
| `↓` / `j` | 选择下一个邮箱 |
| `Tab` | 切换焦点（邮箱列表 ↔ 日志） |
| `PgUp` / `u` | 日志向上滚动 |
| `PgDn` / `d` | 日志向下滚动 |
| `g` | 跳转到日志顶部 |
| `G` | 跳转到日志底部 |

---

## 架构设计

```
email-bot/
├── main.go              # 命令行入口
├── config/
│   └── config.go        # YAML 配置加载与验证
├── core/
│   ├── bot.go           # 调度器、事件总线、并发控制
│   ├── fetcher.go       # IMAP 增量获取（go-imap v1）
│   ├── forwarder.go     # SMTP 转发（STARTTLS / SSL）
│   └── state.go         # 持久化 UID 高水位标记（JSON）
└── tui/
    └── app.go           # Bubbletea TUI（双面板布局）
```

### 事件流程

```
Bot.Run()
  └─ 每隔 poll_interval 秒:
       ├─ pollSource(src1)  ─┐  并发 goroutine
       ├─ pollSource(src2)  ─┤
       └─ pollSource(srcN)  ─┘
            │
            ├─ FetchNewEmails() → IMAP UID 搜索 + RFC-822 下载
            └─ ForwardEmail()   → SMTP 发送（带 Resent-* 头）
                    │
                    └─► Bot.events chan ──► TUI.Update()
```

---

## 跨平台构建

```bash
make build-windows-amd64   # → dist/email-bot-windows-amd64.exe
make build-linux-amd64     # → dist/email-bot-linux-amd64
make build-linux-arm64     # → dist/email-bot-linux-arm64
make build-darwin-amd64    # → dist/email-bot-darwin-amd64   (Intel Mac)
make build-darwin-arm64    # → dist/email-bot-darwin-arm64   (Apple M1/M2/M3)
```

---

## 开源协议

MIT
