# Email Bot

Email Bot 是一个使用 Go 语言编写的轻量级邮件增量拉取与自动转发工具。它采用了 **守护进程 (Daemon)** + **终端图形界面 (TUI)** 的架构设计。支持中文字符集解码，跨平台运行。

## 功能特性
- **定时轮询**：定时从指定的 IMAP 邮箱增量拉取邮件。
- **状态记忆**：本地持久化记录已处理邮件的 UID，重启不会重复发送。
- **多对多转发**：支持灵活的 YAML 路由规则配置，实现一对多、多对多的邮件转发。
- **中文友好**：支持 GBK/GB18030 等中文老式邮件的编码解析。
- **现代化 TUI**：提供基于 `bubbletea` 的高颜值实时监控面板。
- **跨平台**：一键编译出 Windows, Linux, macOS (Intel & M1) 多平台产物。

## 使用方法

### 1. 编译
确保已安装 Go (1.20+)，在项目根目录运行：
```bash
make
```
编译产物将生成在 `build/` 目录下。

### 2. 配置
复制 `config.example.yaml` 为 `config.yaml`，并填写你的邮箱配置：
```yaml
# config.yaml
poll_interval: 60s
api:
  address: "127.0.0.1:8080"
# ... 配置 sources, targets, rules
```

### 3. 运行守护进程 (Daemon)
内核将作为后台进程运行，进行实际的邮件收取和转发：
```bash
./build/email-bot-linux daemon -c config.yaml
```

### 4. 启动监控面板 (TUI)
在另一个终端窗口启动 TUI 客户端以实时查看日志和状态：
```bash
./build/email-bot-linux tui -a 127.0.0.1:8080
```

## 技术栈
- [Cobra](https://github.com/spf13/cobra) - CLI 框架
- [BubbleTea](https://github.com/charmbracelet/bubbletea) - TUI 界面
- [go-imap](https://github.com/emersion/go-imap) - IMAP 邮件拉取
- [gomail.v2](https://gopkg.in/gomail.v2) - SMTP 邮件发送
- [go-message](https://github.com/emersion/go-message) - 邮件结构与字符集解析