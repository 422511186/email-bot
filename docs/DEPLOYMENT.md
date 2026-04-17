# 部署指南

## 环境要求

### 运行时

| 要求 | 说明 |
|------|------|
| Go 版本 | 1.21 或更高 |
| 网络 | 能够访问 IMAP 和 SMTP 服务器 |
| 磁盘 | 约 50MB 用于程序和状态文件 |

### 构建

| 要求 | 说明 |
|------|------|
| Go 工具链 | 1.21+ |
| Git | 用于克隆仓库 |

---

## 安装方式

### 方式一：从源码构建

```bash
# 克隆仓库
git clone https://github.com/yourname/email-bot.git
cd email-bot

# 安装依赖
make deps

# 运行
make run
```

### 方式二：下载预编译二进制

从 [Releases](../../releases) 页面下载对应平台的压缩包，解压后直接运行。

---

## 配置步骤

### 1. 创建配置文件

```bash
# 复制示例配置
cp config.yaml.example config.yaml

# 编辑配置
vim config.yaml
```

### 2. 填写配置

参考 [配置指南](CONFIG.md) 填写：
- 源邮箱（IMAP）凭据
- SMTP 发送凭据
- 目标邮箱地址

### 3. 验证配置

```bash
# 检查配置是否正确
go run . -config config.yaml
```

如果配置正确，会启动 TUI 界面。

---

## 运行方式

### 前台运行（TUI 模式）

```bash
go run . -config config.yaml
```

显示终端用户界面，适合桌面或长期监控。

### 后台运行（守护进程）

**Linux/macOS (systemd)**：

```ini
# /etc/systemd/system/email-bot.service
[Unit]
Description=Email Bot
After=network.target

[Service]
Type=simple
User=youruser
WorkingDirectory=/path/to/email-bot
ExecStart=/path/to/email-bot/email-bot -config /path/to/email-bot/config.yaml
Restart=on-failure
RestartSec=5

[Install]
WantedBy=multi-user.target
```

```bash
sudo systemctl enable email-bot
sudo systemctl start email-bot
sudo journalctl -u email-bot -f  # 查看日志
```

**Windows (任务计划程序)**：

1. 创建批处理文件 `run.bat`：
```batch
@echo off
cd /d D:\path\to\email-bot
email-bot.exe -config config.yaml
```

2. 使用任务计划程序创建任务，设置为"只在登录后运行"

### Docker 运行

```dockerfile
FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY . .
RUN go build -o email-bot .

FROM alpine:latest
RUN apk add --no-cache ca-certificates
COPY --from=builder /app/email-bot /usr/local/bin/
COPY config.yaml.example /app/config.yaml
WORKDIR /app
CMD ["email-bot", "-config", "config.yaml"]
```

```bash
docker build -t email-bot .
docker run -d \
  --name email-bot \
  -v /path/to/config.yaml:/app/config.yaml \
  email-bot
```

---

## 跨平台构建

使用 Makefile 构建各平台二进制：

```bash
# 当前平台
make build

# 所有平台（输出到 dist/）
make build-all

# 单独构建
make build-windows-amd64   # Windows x64
make build-linux-amd64     # Linux x64
make build-linux-arm64     # Linux ARM64
make build-darwin-amd64    # macOS Intel
make build-darwin-arm64    # macOS Apple Silicon
```

---

## 状态文件

状态文件默认位置：`~/.email-bot/state.json`

**手动重置**：
```bash
# 删除状态文件，将从当前位置重新开始
rm ~/.email-bot/state.json
```

**备份/恢复**：
```bash
# 备份
cp ~/.email-bot/state.json state.json.backup

# 恢复
cp state.json.backup ~/.email-bot/state.json
```

---

## 日志查看

TUI 界面实时显示日志。

**守护进程日志** (systemd)：
```bash
journalctl -u email-bot -f
```

**文件日志**（如需要）：
```go
// 可修改代码添加文件日志
log.SetOutput(file)
```

---

## 常见问题

### Q: 如何更新程序？

```bash
git pull
make deps
make build
```

### Q: 如何查看版本？

```bash
./email-bot -version
```

### Q: 如何指定其他配置文件？

```bash
./email-bot -config /path/to/other-config.yaml
```

### Q: 如何停止后台运行？

```bash
# systemd
sudo systemctl stop email-bot

# Docker
docker stop email-bot
```

### Q: 状态文件权限错误？

```bash
mkdir -p ~/.email-bot
chmod 700 ~/.email-bot
```
