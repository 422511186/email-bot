# 📧 Email Bot - Code Wiki

## 1. 项目概述
Email Bot 是一个使用 Go 语言编写的终端邮件转发机器人。它通过轮询多个 IMAP 邮箱，增量获取新邮件，并将其转发到一个或多个目标地址，同时提供了一个基于 Bubbletea 构建的实时 TUI (终端用户界面) 仪表板。

## 2. 项目整体架构
项目采用了清晰的模块化分层架构，主要分为：**入口层**、**配置层**、**核心业务层 (Core)** 和 **展示层 (TUI)**。

### 核心执行流程：
1. **启动阶段**：[main.go](file:///workspace/main.go) 加载 YAML 配置，初始化 [Bot](file:///workspace/core/bot.go#L44-L54) 实例，并启动后台调度协程 (`go bot.Run()`)，最后启动 TUI [Model](file:///workspace/tui/app.go#L96-L110) 接管主线程。
2. **轮询阶段**：[Bot.Run()](file:///workspace/core/bot.go#L121-L142) 根据配置的 `poll_interval` 定期触发轮询。
3. **获取阶段**：[FetchNewEmails()](file:///workspace/core/fetcher.go#L37-L85) 通过 IMAP 协议连接源邮箱，根据本地持久化的 UID 增量拉取新邮件（首次运行仅记录最大UID）。
4. **转发阶段**：拉取到新邮件后，调用 [ForwardEmail()](file:///workspace/core/forwarder.go#L21-L35) 通过 SMTP 将邮件（带上 `Resent-*` 伪装或修改 Subject 头）转发至目标邮箱。
5. **持久化与事件**：更新最大 UID 到 [State](file:///workspace/core/state.go#L17-L21) 并原子性存入本地 `state.json`；同时通过 Event Channel 通知 TUI 层更新界面。

## 3. 主要模块职责

- **[main.go](file:///workspace/main.go)**
  - **职责**：程序入口点，负责解析命令行参数、加载配置、拼装各模块（Bot 引擎和 TUI 界面），并管理生命周期。
- **`config/` (配置解析)**
  - **职责**：加载和验证 `config.yaml`。管理邮箱源 (IMAP)、转发目标、SMTP 配置及全局参数（轮询间隔、延迟等）。
- **`core/` (核心业务逻辑)**
  - **职责**：项目的引擎层，不包含任何 UI 逻辑。负责调度任务、网络通信（IMAP/SMTP）和状态管理。
  - **子模块**：
    - [bot.go](file:///workspace/core/bot.go)：并发调度与事件总线中心。
    - [fetcher.go](file:///workspace/core/fetcher.go)：IMAP 客户端，处理 UID 搜索和邮件原文下载。
    - [forwarder.go](file:///workspace/core/forwarder.go)：SMTP 客户端，处理邮件头的修改（如添加前缀、发件人信息）及邮件发送（支持 STARTTLS / SSL）。
    - [state.go](file:///workspace/core/state.go)：处理 `state.json` 的读写，保存邮箱的 UID 高水位线（High-water mark）以实现增量同步。
- **`tui/` (终端界面)**
  - **职责**：使用 Bubbletea 实现双面板 TUI 界面。左侧显示邮箱状态与进度，右侧展示实时滚动日志。接收后台发出的事件并重新渲染。内部实现在 [app.go](file:///workspace/tui/app.go)。

## 4. 关键类与函数说明

### 4.1 配置管理 (`config` 包)
- **[Config](file:///workspace/config/config.go#L31-L38)**：配置实体结构。其中的 `SourceAccount` 包含单个 IMAP 源及对应的 `Targets`。
- **[Load()](file:///workspace/config/config.go#L40-L90)**：读取并反序列化 YAML，注入默认值（如默认端口 993/587）并执行字段校验。

### 4.2 核心调度 (`core` 包)
- **[Bot](file:///workspace/core/bot.go#L44-L54)**：核心调度器，维护配置、状态、事件通道（`events`）以及各邮箱当前状态的并发安全映射。
- **[Bot.Run()](file:///workspace/core/bot.go#L121-L142)**：阻塞的无限循环函数，根据定时器或手动触发信号（`pollNow`）执行 `runPollCycle()`。
- **[Bot.pollSource()](file:///workspace/core/bot.go#L164-L243)**：处理单一邮箱的具体流程：获取最新邮件 -> 逐封转发 -> 更新内存及事件日志。

### 4.3 邮件获取与转发 (`core` 包)
- **[FetchNewEmails()](file:///workspace/core/fetcher.go#L37-L85)**：
  - 连接 IMAP 服务器，如果是首次初始化则调用 `findMaxUID` 获取最高水位并返回；否则调用 `searchUIDs` 获取新 UID 并通过 `fetchMessages` 下载 RFC-822 原文。
- **[FetchedEmail](file:///workspace/core/fetcher.go#L16-L22)**：承载单封邮件元数据与原始数据的结构体。
- **[ForwardEmail()](file:///workspace/core/forwarder.go#L21-L35)**：
  - 通过 SMTP 协议发送邮件。根据端口判断使用 STARTTLS (`smtp.SendMail`) 还是隐式 TLS (`sendMailImplicitTLS`)。

### 4.4 状态持久化 (`core` 包)
- **[State](file:///workspace/core/state.go#L17-L21)**：包含 `Sources` 映射，记录每个账号的 `LastUID` 和 `Initialized` 标志。
- **[LoadState()](file:///workspace/core/state.go#L28-L39)**：从文件系统加载历史同步进度。
- **[State.Save()](file:///workspace/core/state.go#L42-L60)**：将当前高水位状态通过临时文件重命名的方式**原子性**地写入磁盘，防止数据损坏。

### 4.5 界面展示 (`tui` 包)
- **[Model](file:///workspace/tui/app.go#L96-L110)**：Bubbletea 的核心模型，维护界面宽高、列表选中索引、日志数组及滚动偏移量。
- **[NewModel()](file:///workspace/tui/app.go#L112-L119)**：构造并初始化 TUI 数据模型。
- **[Model.Update()](file:///workspace/tui/app.go#L130-L166)**：状态机核心，处理按键事件（`tea.KeyMsg`）、定时器 (`tickMsg`) 以及后台传来的 `botEventMsg`。

## 5. 依赖关系

项目主要依赖以下优秀的开源 Go 库：
- **UI 与终端**：
  - [`github.com/charmbracelet/bubbletea`](https://github.com/charmbracelet/bubbletea): 核心 TUI 框架，基于 The Elm Architecture。
  - [`github.com/charmbracelet/lipgloss`](https://github.com/charmbracelet/lipgloss): 终端样式与布局渲染。
- **邮件协议处理**：
  - [`github.com/emersion/go-imap`](https://github.com/emersion/go-imap): IMAP 客户端库，用于连接邮箱及搜索 UID。
  - [`github.com/emersion/go-message`](https://github.com/emersion/go-message): 解析和处理邮件编码与内容。
- **配置与工具**：
  - [`gopkg.in/yaml.v3`](https://gopkg.in/yaml.v3): YAML 配置文件解析。
  - [`golang.org/x/text`](https://golang.org/x/text): 处理中文字符集编码（GBK, GB18030 等）。

## 6. 项目运行方式

### 6.1 前置要求
- Go 1.21 或更高版本

### 6.2 编译与构建
项目提供了一个 [Makefile](file:///workspace/Makefile) 简化操作：
```bash
# 安装依赖
make deps

# 编译当前平台版本
make build

# 交叉编译全平台版本
make build-all
```

### 6.3 配置与运行
1. 生成配置文件：
   ```bash
   make init-config
   # 或手动复制：cp config.yaml.example config.yaml
   ```
2. 编辑 `config.yaml` 填入 IMAP 源邮箱及 SMTP 转发账号的凭据（注意使用应用密码/授权码）。
3. 运行程序：
   ```bash
   # 开发环境运行
   make run
   # 或使用构建产物
   ./email-bot -config config.yaml
   ```

### 6.4 TUI 交互方式
- `↑ / ↓` 或 `k / j`：切换左侧查看的邮箱源
- `Tab`：在左侧邮箱列表与右侧日志面板间切换焦点
- `PgUp / PgDn` 或 `u / d`：滚动查看历史活动日志
- `r`：立即触发一次所有邮箱的轮询拉取
- `q` 或 `Ctrl+C`：安全退出程序
