# 配置指南

## 完整配置示例

```yaml
# ================================================================
# Email Bot 配置
# ================================================================

# 轮询每个源邮箱的间隔（秒）
poll_interval: 120

# 转发邮件之间的间隔（毫秒），避免触发目标邮箱限流
# 默认为 1000（1秒）
forward_delay: 1000

# ================================================================
# 源邮箱配置（IMAP）
# ================================================================
sources:
  # 示例 1: Gmail → 两个目标（一对多）
  - name:     "工作邮箱"
    host:     imap.gmail.com
    port:     993          # 默认 993（TLS）
    username: your@gmail.com
    password: "xxxx xxxx xxxx xxxx"   # Gmail 应用密码（16位）
    mailbox:  INBOX        # 默认 INBOX
    targets:
      - archive@company.com
      - notifications@company.com

  # 示例 2: QQ 邮箱 → 单个目标
  - name:     "个人邮箱"
    host:     imap.qq.com
    port:     993
    username: 123456789@qq.com
    password: "your_qq_auth_code"     # QQ 授权码
    mailbox:  INBOX
    targets:
      - personal@example.com

# ================================================================
# SMTP 配置（发送转发的邮件）
# ================================================================
smtp:
  host:     smtp.gmail.com
  port:     587            # 587 = STARTTLS  |  465 = 隐式 SSL
  username: sender@gmail.com
  password: "xxxx xxxx xxxx xxxx"
  from:     sender@gmail.com   # 默认使用 username

# ================================================================
# 状态文件（可选）
# ================================================================
# 追踪每个邮箱的最后已见 UID
# 默认为 ~/.email-bot/state.json
# state_file: /自定义路径/state.json
```

---

## 配置项详解

### poll_interval

**类型**：`int`  
**默认值**：`60`  
**单位**：秒

轮询间隔，控制多久检查一次新邮件。

| 值 | 效果 |
|----|------|
| 30 | 每 30 秒检查一次（频繁） |
| 60 | 每 1 分钟检查一次（推荐） |
| 300 | 每 5 分钟检查一次（省资源） |
| 600 | 每 10 分钟检查一次（低频） |

**注意**：过短的间隔可能触发邮箱服务商的限流。

### forward_delay

**类型**：`int`  
**默认值**：`1000`  
**单位**：毫秒

每封邮件转发之间的延迟，避免目标邮箱被识别为垃圾邮件。

| 值 | 效果 |
|----|------|
| 500 | 每封间隔 0.5 秒（快速） |
| 1000 | 每封间隔 1 秒（默认） |
| 2000 | 每封间隔 2 秒（稳健） |
| 5000 | 每封间隔 5 秒（保守） |

**适用场景**：
- 批量转发多封邮件时调大此值
- 目标邮箱经常拒绝接收时调大此值

---

## SourceAccount 配置

### name

**类型**：`string`  
**默认值**：`username`（即邮箱地址）

显示名称，用于 TUI 界面显示。

```yaml
name: "工作邮箱"      # 推荐使用有意义的名字
```

### host

**类型**：`string`  
**必填**：是

IMAP 服务器主机地址。

```yaml
host: imap.gmail.com   # Gmail
host: imap.qq.com      # QQ 邮箱
host: imap.outlook.com # Outlook
```

### port

**类型**：`int`  
**默认值**：`993`

IMAP 端口号。

| 端口 | 加密 | 说明 |
|------|------|------|
| 993 | TLS | 推荐，隐式 TLS |
| 143 | 无/STARTTLS | 明文或升级加密 |

### username

**类型**：`string`  
**必填**：是

完整邮箱地址。

```yaml
username: your@gmail.com
username: 123456@qq.com
```

### password

**类型**：`string`  
**必填**：是

应用密码或授权码，**不是登录密码**。

| 邮箱 | 密码类型 | 获取方式 |
|------|----------|----------|
| Gmail | 应用密码 | Google 账户 → 安全性 → 应用密码 |
| QQ 邮箱 | 授权码 | QQ 邮箱 → 设置 → 账户 → IMAP/SMTP |
| Outlook | 应用密码 | Microsoft 账户 → 安全 → 应用密码 |

### mailbox

**类型**：`string`  
**默认值**：`INBOX`

邮箱文件夹名称，通常不需要修改。

```yaml
mailbox: INBOX        # 收件箱
mailbox: "[Gmail]/已归档"  # Gmail 归档文件夹
```

### targets

**类型**：`[]string`  
**必填**：是

目标邮箱地址列表，一封邮件会转发到所有目标。

```yaml
targets:
  - archive@company.com
  - backup@example.com
  - notify@company.com
```

---

## SMTPConfig 配置

### host

**类型**：`string`  
**必填**：是

SMTP 服务器主机地址。

```yaml
host: smtp.gmail.com   # Gmail
host: smtp.qq.com      # QQ 邮箱
host: smtp.outlook.com # Outlook
```

### port

**类型**：`int`  
**默认值**：`587`

SMTP 端口号。

| 端口 | 加密方式 | 说明 |
|------|----------|------|
| 587 | STARTTLS | 推荐，先明文后升级 |
| 465 | 隐式 TLS | 直接加密连接 |
| 25 | 无/STARTTLS | 大多数被封禁，不推荐 |

### username / password

**类型**：`string`  
**必填**：是

SMTP 认证凭据，通常与 IMAP 相同。

### from

**类型**：`string`  
**默认值**：`smtp.username`

信封发件人地址（MAIL FROM），显示在邮件头部的 `From:`。

---

## 常见邮箱配置

### Gmail

```yaml
sources:
  - name:     "Gmail"
    host:     imap.gmail.com
    port:     993
    username: your@gmail.com
    password: "xxxx xxxx xxxx xxxx"  # 应用密码
    targets:
      - target@example.com

smtp:
  host:     smtp.gmail.com
  port:     587
  username: your@gmail.com
  password: "xxxx xxxx xxxx xxxx"
  from:     your@gmail.com
```

**配置步骤**：
1. 启用 IMAP：设置 → 查看所有设置 → 转发和 POP/IMAP → 启用 IMAP
2. 创建应用密码：myaccount.google.com → 安全性 → 两步验证 → 应用密码

### QQ 邮箱

```yaml
sources:
  - name:     "QQ邮箱"
    host:     imap.qq.com
    port:     993
    username: 123456789@qq.com
    password: "abcdefghijklmn"  # 16位授权码
    targets:
      - target@example.com

smtp:
  host:     smtp.qq.com
  port:     587
  username: 123456789@qq.com
  password: "abcdefghijklmn"
  from:     123456789@qq.com
```

**配置步骤**：
1. 开启 IMAP/SMTP：设置 → 账户 → POP3/IMAP/SMTP/Exchange/CardDAV/CalDAV服务 → 开启 IMAP/SMTP
2. 获取授权码

### Outlook/Microsoft 365

```yaml
sources:
  - name:     "Outlook"
    host:     outlook.office365.com
    port:     993
    username: your@outlook.com
    password: "xxxx xxxx xxxx xxxx"  # 应用密码
    targets:
      - target@example.com

smtp:
  host:     smtp-mail.outlook.com
  port:     587
  username: your@outlook.com
  password: "xxxx xxxx xxxx xxxx"
  from:     your@outlook.com
```

### 网易邮箱（163/126）

```yaml
sources:
  - name:     "163邮箱"
    host:     imap.163.com
    port:     993
    username: yourname@163.com
    password: "YOUR_AUTH_CODE"  # 授权码
    targets:
      - target@example.com

smtp:
  host:     smtp.163.com
  port:     465
  username: yourname@163.com
  password: "YOUR_AUTH_CODE"
  from:     yourname@163.com
```

**注意**：网易邮箱 SMTP 默认使用 465 端口（SSL）。

---

## 故障排除

### 连接失败

**错误**：`connection refused` 或 `timeout`

**排查**：
1. 检查 `host` 和 `port` 是否正确
2. 检查网络连接
3. 检查防火墙/代理设置
4. 确认邮箱服务商的 IMAP 是否已开启

### 认证失败

**错误**：`authentication failed`

**排查**：
1. 确认使用的是**应用密码**而非登录密码
2. Gmail：确认两步验证已开启
3. QQ 邮箱：确认使用的是授权码而非 QQ 密码
4. 检查密码是否正确（无空格）

### 邮件发送失败

**错误**：`smtp server requires authentication`

**排查**：
1. 确认 SMTP `username` 和 `password` 正确
2. 确认 `port` 正确（587 用 STARTTLS，465 用 SSL）
3. 部分邮箱需要开启 SMTP 服务

### 被识别为垃圾邮件

**现象**：转发到目标邮箱的邮件被标记为垃圾邮件

**解决方案**：
1. 调大 `forward_delay`（如设为 2000）
2. 在目标邮箱添加发件人为联系人
3. 确认 `from` 地址与目标邮箱服务商一致
4. 避免短时间内大量转发

### 重复转发

**现象**：同一封邮件被转发多次

**排查**：
1. 检查状态文件是否被意外删除或损坏
2. 确认没有多个实例同时运行
3. 检查 `state_file` 路径是否正确
