# 🔄 Email Bot 优雅升级指南

随着 Email Bot 的不断迭代（如：新增网络超时控制、SMTP 连接复用、MIME 解析修复等重大重构），你需要定期更新服务。

为了保证**邮件不丢失、状态不损坏、服务零感知中断**，请务必按照以下“优雅升级”流程进行源代码的重新编译与替换。

---

## 1. 准备工作与环境检查

在开始升级前，请确保你所在的编译环境满足以下条件：

- **Go 环境**：确保已安装 Go 1.21 或更高版本。
- **Git**：确保可以拉取最新代码。

```bash
go version
git --version
```

## 2. 拉取最新源代码并编译

在不影响当前正在运行的服务的前提下，先在一个新的目录或当前目录克隆/更新代码，并完成编译测试。

```bash
# 1. 进入你的 email-bot 源码目录
cd /path/to/email-bot

# 2. 拉取最新代码
git pull origin main

# 3. 下载/更新最新的依赖包
make deps

# 4. 编译新版本的二进制文件（编译过程不会影响旧版本运行）
make build

# 5. 验证新二进制版本是否成功生成
./email-bot -version
```

此时，你的目录中应该已经生成了名为 `email-bot`（或 `email-bot.exe`）的最新二进制文件。

## 3. 备份核心数据（极其重要）

在替换旧服务前，**必须备份配置文件和状态文件**。状态文件（`state.json`）记录了所有邮箱的增量拉取进度（高水位 UID），一旦丢失，会导致所有历史邮件被重新转发！

```bash
# 1. 备份配置文件
cp config.yaml config.yaml.bak_$(date +%F)

# 2. 备份状态文件（默认路径为 ~/.email-bot/state.json）
# 注意：如果你的 config.yaml 中自定义了 state_file 路径，请备份对应路径
cp ~/.email-bot/state.json ~/.email-bot/state.json.bak_$(date +%F)
```

## 4. 优雅停机 (Graceful Shutdown)

最新版本的 Email Bot 已经支持了“优雅停机”。这意味着它会等待当前正在转发的邮件处理完毕，并将最新的 UID 安全落盘后才会退出，避免强制中断导致的邮件丢失或状态不一致。

### 方式一：如果是在 TUI 终端前台运行
直接在终端中按下 `q` 或 `Ctrl+C`。
你会看到日志面板提示：`🛑 邮件机器人正在安全停止中...`。请耐心等待程序完全退出回到 Shell 提示符。

### 方式二：如果是在后台运行 (如 Systemd 或 Nohup)
发送 `SIGTERM` 或 `SIGINT` 信号给进程（不要使用 `kill -9`）：

```bash
# 查找进程 ID
pgrep email-bot

# 发送优雅停止信号
kill -15 <PID>
```
*提示：如果是 Systemd 管理的服务，直接执行 `sudo systemctl stop email-bot` 即可。*

## 5. 替换与重启

确认旧进程已经完全退出后，使用刚刚编译好的新版本覆盖旧版本，并重新启动。

```bash
# 如果你是将二进制放在了系统 PATH 中（如 /usr/local/bin）
# sudo cp ./email-bot /usr/local/bin/email-bot

# 启动新版本服务
# 前台运行查看状态：
./email-bot -config config.yaml

# 或者使用 Systemd 重启：
# sudo systemctl start email-bot
```

## 6. 验证升级结果

升级完成后，请通过以下几个维度确认服务是否正常运转：

1. **版本确认**：
   观察 TUI 面板或使用 `./email-bot -version` 确认运行的是新版本。
2. **状态继承检查**：
   观察 TUI 左侧的邮箱列表，确认各个邮箱的“上次轮询时间”和图标（`✓`）是否正常，且没有触发历史邮件的全量重发。
3. **功能验证**：
   - 按下 `r` 键手动触发一次轮询。
   - 观察右侧的“活动日志”，确认是否输出了 `✅ 轮询周期完成`。
   - 检查目标邮箱，发送一封测试邮件到源邮箱，确认是否能在几秒内被正确转发（且包含 `[原发件人] - ` 的正确前缀）。

---

## 常见问题 (FAQ)

**Q1：升级后报错 `解析状态: json: cannot unmarshal...` 怎么办？**
A：这通常是因为跨大版本升级导致 `state.json` 结构不兼容。请停止服务，将备份的 `.bak` 文件恢复，或删除 `state.json` 让机器人重新初始化（注意：删除会导致重新标记高水位，在此之前的未读邮件将不会被转发）。

**Q2：编译时提示 `go: downloading... connection refused`？**
A：请检查你的网络，或配置 Go 模块代理：
```bash
go env -w GOPROXY=https://goproxy.cn,direct
make deps
```

**Q3：新的配置文件项怎么更新？**
A：对比新源码中的 `config.yaml.example` 和你现有的 `config.yaml`。由于向后兼容设计，通常旧配置依然可以运行。如果有新增的必填项，程序会在启动时在终端打印 `❌ 加载配置失败` 的明确提示，根据提示添加即可。