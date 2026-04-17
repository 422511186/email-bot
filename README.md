# 🤖 Email Bot

Email Bot 是一个使用 Go 语言编写的轻量级、跨平台的邮件增量拉取与自动转发工具。它采用了 **守护进程 (Daemon)** + **终端图形界面 (TUI)** 的分离架构设计。支持复杂的路由规则、中文字符集解码，并能通过高颜值的终端界面实时监控运行状态。

---

## ✨ 核心特性

- **🚀 增量拉取与状态记忆**：内核使用 IMAP 协议定时轮询，拉取新邮件后会将每个邮箱已处理的最高 UID 原子性写入本地 `state.json` 中。即使程序重启或崩溃，也能从断点处继续拉取，彻底避免重复转发。
- **🔀 灵活的路由规则 (多对多 / 一对多)**：支持通过 YAML 配置文件设定 `rules` 列表，轻松实现将多个邮箱的新邮件汇总到一个邮箱，或者将一个邮箱的新邮件群发给多个目标邮箱。
- **🇨🇳 完美的中文兼容**：内置对早期国内邮件系统常有的 GBK/GB18030/GB2312 字符集解码支持，自动将其转换为 UTF-8，彻底解决中文邮件乱码问题。
- **💻 现代化 TUI 面板**：提供基于 `bubbletea` 的高颜值实时监控终端面板，支持全角中文字符对齐。
- **🛠️ 跨平台编译**：提供一键构建脚本，支持 Windows、Linux 以及 macOS (包含 M 系列芯片和 Intel 芯片)。

---

## 📦 快速开始

### 1. 编译安装

确保你的系统已安装 [Go (1.20+)](https://golang.org/dl/)，然后在项目根目录运行：

```bash
make
```

编译完成后，你会在 `build/` 目录下看到针对不同平台的二进制可执行文件：
- `email-bot.exe` (Windows amd64)
- `email-bot-linux` (Linux amd64)
- `email-bot-mac-intel` (macOS amd64)
- `email-bot-mac-m1` (macOS arm64)

> ⚠️ **系统兼容性**：预编译的 Linux 二进制文件依赖于 `GLIBC_2.34`。如果运行时报错 `version 'GLIBC_2.34' not found`，请使用 `go build` 重新编译：
> ```bash
> go build -o build/email-bot-linux ./main.go
> ```

### 2. 配置文件说明

在运行前，需要配置邮箱的账号和路由规则。项目中提供了一个示例配置文件 `config.example.yaml`。你可以将其复制并重命名为 `config.yaml`。

配置项详解：

```yaml
# 全局轮询间隔时间 (例如: 60s, 5m, 1h)
poll_interval: 60s

# Daemon 的内部 HTTP API 监听地址，供 TUI 客户端调用
api:
  address: "127.0.0.1:8080"

# 需要监听并拉取邮件的源邮箱列表
sources:
  - id: "source1"                  # 唯一标识符
    host: "imap.qq.com:993"        # IMAP 服务器地址及端口
    username: "user@qq.com"        # 邮箱账号
    password: "your-auth-code"     # IMAP 授权码 (通常不是登录密码)
  - id: "source2"
    host: "imap.163.com:993"
    username: "user@163.com"
    password: "your-auth-code"

# 目标邮箱列表（用于发送转发邮件的 SMTP 服务器及账号）
targets:
  - id: "target1"
    host: "smtp.qq.com:465"        # SMTP 服务器地址及端口
    username: "forward@qq.com"     # 发信账号
    password: "your-auth-code"     # SMTP 授权码
    email: "forward@qq.com"        # 实际的 "发件人" 邮箱地址
  - id: "target2"
    host: "smtp.gmail.com:465"
    username: "forward@gmail.com"
    password: "your-auth-code"
    email: "forward@gmail.com"

# 转发路由规则
rules:
  # 规则 1：将 source1 收到的新邮件，同时转发给 target1 和 target2（一对多）
  - id: "rule1"
    sources:
      - "source1"
    targets:
      - "target1"
      - "target2"
  
  # 规则 2：将 source1 和 source2 的新邮件，统一汇总转发给 target1（多对一 / 多对多）
  - id: "rule2"
    sources:
      - "source1"
      - "source2"
    targets:
      - "target1"
```

> **注意：** 出于安全考虑，主流邮箱（如 QQ、网易、Gmail）均要求使用**独立授权码**而非网页登录密码进行 IMAP/SMTP 访问，请在各邮箱的网页端设置中生成并获取。

### 3. 运行守护进程 (Daemon)

Daemon 模式是程序的核心，它在后台运行，负责实际的邮件收取、解析、状态记录和转发。

```bash
# 以 Linux 平台为例，指定配置文件路径启动
./build/email-bot-linux daemon -c config.yaml
```

*可选参数：*
- `-c, --config`: 指定 YAML 配置文件路径（默认为 `config.yaml`）
- `-s, --state`: 指定本地状态 JSON 文件路径（默认为 `state.json`，程序会自动创建和更新）

### 4. 启动 TUI 监控面板

在 Daemon 正常运行后，你可以随时在另一个终端窗口启动 TUI 客户端。TUI 客户端会通过本地 HTTP 接口获取 Daemon 的实时状态和运行日志。

```bash
./build/email-bot-linux tui -a 127.0.0.1:8080
```

*快捷键：*
- 按 `q` 或 `Ctrl+C` 退出 TUI 面板。退出 TUI **不会**影响后台 Daemon 进程的运行。

---

## 🏗️ 架构与工作原理

1. **分离架构**：程序分为 `daemon` (服务端) 和 `tui` (客户端) 两个子命令。`daemon` 进程暴露了一个轻量级的 HTTP REST API (`/api/status`, `/api/logs`) 供客户端查询。
2. **增量拉取 (UID)**：IMAP 协议为每封邮件分配了唯一的 UID。程序首次运行时，如果没有历史记录，默认会拉取收件箱中最新的邮件并记录最高 UID；之后的每一次轮询，都会使用 `UID SEARCH` 指令查找大于记录 UID 的新邮件，从而实现精准的增量拉取。
3. **中文解码处理**：通过引入 `golang.org/x/text/encoding/simplifiedchinese` 并在 `go-message` 中注册自定义的字符集处理器，Email Bot 可以无缝兼容并解析包含 `GBK` 或 `GB18030` 编码的国内老式邮件，并在转发时将其统一转换为标准的 `UTF-8` 编码。

---

## 📚 技术栈

- [Cobra](https://github.com/spf13/cobra) - 强大的现代 CLI 命令行应用构建框架
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - 流行的 Elm 架构 TUI (Terminal User Interface) 框架
- [go-imap](https://github.com/emersion/go-imap) - IMAP 客户端库，处理邮件拉取
- [gomail.v2](https://gopkg.in/gomail.v2) - SMTP 客户端库，处理邮件发送
- [go-message](https://github.com/emersion/go-message) - 用于解析复杂的 MIME 邮件结构和字符集

---

## 📜 许可证

本项目基于 MIT License 开源。