package config

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"gopkg.in/yaml.v3"
)

// SourceAccount 是监控的邮箱及其转发目标。
type SourceAccount struct {
	Name     string   `yaml:"name"`     // 显示名称
	Host     string   `yaml:"host"`    // IMAP 主机，例如 imap.gmail.com
	Port     int      `yaml:"port"`    // 默认 993（TLS）
	Username string   `yaml:"username"` // 完整邮箱地址
	Password string   `yaml:"password"` // 应用密码 / 授权码
	Mailbox  string   `yaml:"mailbox"` // 默认 INBOX
	Targets  []string `yaml:"targets"` // 一个或多个目标地址
}

// SMTPConfig 是用于转发邮件的发送邮件服务器配置。
type SMTPConfig struct {
	Host     string `yaml:"host"`
	Port     int    `yaml:"port"`     // 587 (STARTTLS) 或 465 (SSL)
	Username string `yaml:"username"`
	Password string `yaml:"password"`
	From     string `yaml:"from"` // 信封发件人地址
}

// Config 是根配置结构。
type Config struct {
	PollInterval int             `yaml:"poll_interval"`  // 轮询间隔秒数，默认 60
	ForwardDelay int             `yaml:"forward_delay"`  // 转发邮件间隔（毫秒），默认 1000
	Sources      []SourceAccount `yaml:"sources"`
	SMTP         SMTPConfig      `yaml:"smtp"`
	StateFile    string          `yaml:"state_file"` // 默认 ~/.email-bot/state.json
}

// Load 读取并验证 YAML 配置文件。
func Load(path string) (*Config, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("读取配置文件: %w", err)
	}

	cfg := &Config{
		PollInterval: 60,
		ForwardDelay: 1000,
	}
	if err := yaml.Unmarshal(data, cfg); err != nil {
		return nil, fmt.Errorf("解析配置: %w", err)
	}

	// ── 应用默认值 ───────────────────────────────────────────
	for i := range cfg.Sources {
		s := &cfg.Sources[i]
		// Normalize user input early to avoid confusing IMAP errors.
		s.Host = strings.TrimSpace(s.Host)
		s.Username = strings.TrimSpace(s.Username)
		s.Password = strings.TrimSpace(s.Password)
		s.Mailbox = strings.TrimSpace(s.Mailbox)

		if s.Port == 0 {
			s.Port = 993
		}
		if s.Mailbox == "" {
			s.Mailbox = "INBOX"
		}
		// Common typo: INB0X (zero) instead of INBOX (letter O).
		if strings.EqualFold(s.Mailbox, "INB0X") {
			s.Mailbox = "INBOX"
		}
		if s.Name == "" {
			s.Name = s.Username
		}

		for ti := range s.Targets {
			s.Targets[ti] = strings.TrimSpace(s.Targets[ti])
		}
	}

	if cfg.SMTP.Port == 0 {
		cfg.SMTP.Port = 587
	}
	cfg.SMTP.Host = strings.TrimSpace(cfg.SMTP.Host)
	cfg.SMTP.Username = strings.TrimSpace(cfg.SMTP.Username)
	cfg.SMTP.Password = strings.TrimSpace(cfg.SMTP.Password)
	if cfg.SMTP.From == "" {
		cfg.SMTP.From = cfg.SMTP.Username
	}
	cfg.SMTP.From = strings.TrimSpace(cfg.SMTP.From)

	if cfg.StateFile == "" {
		home, err := os.UserHomeDir()
		if err != nil {
			home = "."
		}
		cfg.StateFile = filepath.Join(home, ".email-bot", "state.json")
	}

	// ── 验证 ─────────────────────────────────────────────────
	if err := validate(cfg); err != nil {
		return nil, err
	}

	return cfg, nil
}

func validate(cfg *Config) error {
	if len(cfg.Sources) == 0 {
		return fmt.Errorf("配置: 未定义源账户")
	}
	if cfg.SMTP.Host == "" {
		return fmt.Errorf("配置: smtp.host 为必填项")
	}
	if cfg.SMTP.Username == "" {
		return fmt.Errorf("配置: smtp.username 为必填项")
	}
	for i, src := range cfg.Sources {
		if src.Host == "" {
			return fmt.Errorf("配置: sources[%d].host 为必填项", i)
		}
		if src.Username == "" {
			return fmt.Errorf("配置: sources[%d].username 为必填项", i)
		}
		if len(src.Targets) == 0 {
			return fmt.Errorf("配置: sources[%d] (%s) 没有定义目标", i, src.Username)
		}
	}
	return nil
}
