# Code Wiki - 文档目录

欢迎使用 Email Bot 项目文档中心。

## 📚 文档索引

### [项目概述](OVERVIEW.md)
- 项目简介与背景
- 核心功能特性
- 技术选型理由

### [架构设计](ARCHITECTURE.md)
- 整体系统架构
- 模块依赖关系
- 事件流程详解
- 数据流向图

### [模块详解](MODULES.md)
- [main.go](../main.go) - 程序入口
- [core/bot.go](../core/bot.go) - 机器人核心引擎
- [core/fetcher.go](../core/fetcher.go) - IMAP 邮件获取
- [core/forwarder.go](../core/forwarder.go) - SMTP 邮件转发
- [core/state.go](../core/state.go) - 状态持久化
- [config/config.go](../config/config.go) - 配置管理
- [tui/app.go](../tui/app.go) - 终端用户界面

### [API 参考](API.md)
- 核心类型定义
- 函数接口说明
- 事件类型列表
- 配置结构详解

### [配置指南](CONFIG.md)
- 完整配置示例
- 各配置项详解
- 常见邮箱配置
- 故障排除

### [部署指南](DEPLOYMENT.md)
- 环境要求
- 安装步骤
- 运行方式
- 跨平台构建

## 🔗 快速链接

- [项目 README](../README.md) - 快速开始
- [配置示例](../config.yaml.example) - 配置文件参考
- [Makefile](../Makefile) - 构建命令

## 📝 文档贡献

本文档使用 Markdown 编写，可直接在 GitHub/Gitea 上编辑。如有疑问请提交 Issue。
