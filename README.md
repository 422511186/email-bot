# 📧 Email Bot

一个使用 Go 语言编写的**终端邮件转发机器人**。轮询多个 IMAP 邮箱，增量获取新邮件，并将其转发到一个或多个目标地址 —— 全部通过实时 TUI 仪表板展示。

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

## ✨ 功能特性

| 功能 | 说明 |
|------|------|
| **增量拉取** | 追踪每个邮箱的已见最大 IMAP UID；仅获取新邮件 |
| **首次运行保护** | 首次启动时记录当前邮箱状态，不转发历史邮件 |
| **一对多转发** | 一个源邮箱 → 多个目标地址 |
| **多对多配置** | 多个源邮箱，每个可配置独立的目标列表 |
| **SMTP 传输** | 支持 STARTTLS（587 端口）和隐式 TLS/SSL（465 端口） |
| **邮件间隔控制** | 可配置转发间隔，避免触发目标邮箱限流 |
| **持久化状态** | 重启后状态保留（`~/.email-bot/state.json`） |
| **实时 TUI** | 双面板仪表板：左侧邮箱状态，右侧可滚动活动日志 |
| **跨平台** | Windows、Linux (amd64/arm64)、macOS (Intel + M系列) |
| **附件保留** | 完整保留邮件附件和内联图片 |

---

## 🚀 快速开始

### 前置要求

- **Go 1.21** 或更高版本

```bash
go version
```

### 安装

#### 方式一：从源码构建

```bash
# 克隆仓库
git clone https://github.com/yourname/email-bot.git
cd email-bot

# 安装依赖
make deps

# 构建
make build
```

#### 方式二：下载预编译二进制

从 [Releases](https://github.com/yourname/email-bot/releases) 下载对应平台的压缩包。

### 配置

```bash
# 复制示例配置
cp config.yaml.example config.yaml

# 编辑配置
vim config.yaml   # 或使用其他编辑器
```

#### 最小配置示例

```yaml
poll_interval: 60
forward_delay: 1000

sources:
  - name:     "我的邮箱"
    host:     imap.gmail.com
    port:     993
    username: your@gmail.com
    password: "your-app-password"
    targets:
      - target@example.com

smtp:
  host:     smtp.gmail.com
  port:     587
  username: your@gmail.com
  password: "your-app-password"
  from:     your@gmail.com
```

详细配置说明请参考 [配置指南](docs/CONFIG.md)。

### 运行

```bash
# 方式一：使用 Makefile
make run

# 方式二：直接运行
go run . -config config.yaml

# 方式三：使用构建好的二进制
./email-bot -config config.yaml
```

### Gmail 配置步骤

1. 启用 IMAP：Gmail → 设置 → 查看所有设置 → 转发和 POP/IMAP → 启用 IMAP
2. 创建应用密码：
   - 访问 [Google 账户安全设置](https://myaccount.google.com/security)
   - 开启"两步验证"
   - 生成"应用密码"（选择"其他"→ 输入名称）
   - 使用生成的 16 位应用密码填入配置

### QQ 邮箱配置步骤

1. 开启 IMAP/SMTP：QQ 邮箱 → 设置 → 账户 → POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务
2. 获取授权码（16 位）
3. 使用授权码填入配置

---

## 📖 TUI 操作说明

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

### 状态图标说明

| 图标 | 颜色 | 含义 |
|------|------|------|
| `⟳` | 黄色 | 正在轮询 |
| `✗` | 红色 | 上次轮询失败 |
| `✓` | 绿色 | 正常（已同步） |
| `○` | 灰色 | 等待首次轮询 |

---

## 📂 项目结构

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
├── tui/
│   └── app.go           # Bubbletea TUI（双面板布局）
├── docs/                # 项目文档
│   ├── SUMMARY.md       # 文档目录
│   ├── OVERVIEW.md      # 项目概述
│   ├── ARCHITECTURE.md  # 架构设计
│   ├── MODULES.md       # 模块详解
│   ├── API.md           # API 参考
│   ├── CONFIG.md        # 配置指南
│   └── DEPLOYMENT.md    # 部署指南
├── Makefile             # 构建脚本
└── config.yaml.example  # 配置示例
```

---

## 🏗️ 架构设计

### 核心模块

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

### UID 高水位机制

```
首次运行：
  → 扫描邮箱获取当前最大 UID = 7
  → 记录状态，不转发任何邮件

后续运行（UID 8, 9 到达）：
  → 查找 UID > 7 的邮件
  → 获取并转发 UID 8, 9
  → 更新最大 UID = 9
```

详细架构说明请参考 [架构文档](docs/ARCHITECTURE.md)。

---

## 🔧 Makefile 命令

```bash
make deps           # 安装依赖
make run            # 运行
make build          # 构建当前平台
make build-all      # 构建所有平台
make build-windows-amd64   # Windows x64
make build-linux-amd64    # Linux x64
make build-linux-arm64    # Linux ARM64
make build-darwin-amd64   # macOS Intel
make build-darwin-arm64   # macOS Apple Silicon
make init-config    # 复制配置示例
```

---

## ⚙️ 配置参考

完整配置说明请参考 [配置指南](docs/CONFIG.md)。

### 常用 IMAP 主机

| 邮箱 | IMAP 主机 | 端口 |
|------|-----------|------|
| Gmail | imap.gmail.com | 993 |
| Outlook | outlook.office365.com | 993 |
| QQ 邮箱 | imap.qq.com | 993 |
| 163 邮箱 | imap.163.com | 993 |
| 126 邮箱 | imap.126.com | 993 |
| 新浪邮箱 | imap.sina.com | 993 |
| iCloud | imap.mail.me.com | 993 |

---

## 📚 文档索引

- [项目概述](docs/OVERVIEW.md) - 项目简介与核心功能
- [架构设计](docs/ARCHITECTURE.md) - 系统架构与模块设计
- [模块详解](docs/MODULES.md) - 各模块详细说明
- [API 参考](docs/API.md) - 类型定义与函数接口
- [配置指南](docs/CONFIG.md) - 配置项详解与常见邮箱配置
- [部署指南](docs/DEPLOYMENT.md) - 安装部署与后台运行

---

## 🔒 安全说明

1. **凭据保护**：不要将 `config.yaml` 提交到版本控制系统
2. **文件权限**：状态文件包含邮箱凭据，建议设置适当权限
3. **网络安全**：确保 IMAP/SMTP 连接使用 TLS 加密

---

## 🐛 故障排除

### 连接失败

- 检查 `host` 和 `port` 是否正确
- 确认邮箱服务商的 IMAP 已开启
- 检查防火墙设置

### 认证失败

- Gmail：确认已开启两步验证，使用应用密码
- QQ 邮箱：使用授权码而非 QQ 密码

### 邮件被标记为垃圾邮件

- 调大 `forward_delay`（如设为 2000）
- 在目标邮箱添加发件人为联系人

详细排查请参考 [配置指南 - 故障排除](docs/CONFIG.md#故障排除)。

---

## 📄 开源协议

MIT License
