package core

import (
	"crypto/tls"
	"fmt"
	"net/smtp"
	"strings"
	"time"

	"email-bot/config"
)

// ForwardEmail 通过配置的 SMTP 服务器将邮件的原始 RFC-822 数据重新发送到所有目标地址。
//
// 465 端口 → 隐式 TLS（SSL）
// 587 端口 → STARTTLS（大多数邮件服务商）
// 其他端口 → 纯文本 / 通过 smtp.SendMail 的 STARTTLS
func ForwardEmail(smtpCfg config.SMTPConfig, email FetchedEmail, targets []string) error {
	if len(targets) == 0 {
		return fmt.Errorf("未提供目标地址")
	}

	body := prependResentHeaders(smtpCfg.From, targets, email.Raw)
	addr := fmt.Sprintf("%s:%d", smtpCfg.Host, smtpCfg.Port)
	auth := smtp.PlainAuth("", smtpCfg.Username, smtpCfg.Password, smtpCfg.Host)

	if smtpCfg.Port == 465 {
		return sendMailImplicitTLS(addr, smtpCfg.Host, auth, smtpCfg.From, targets, body)
	}

	// 587 / 25 → smtp.SendMail 在支持时自动升级到 STARTTLS。
	return smtp.SendMail(addr, auth, smtpCfg.From, targets, body)
}

// prependResentHeaders 在原始邮件正文前插入 RFC-2822 Resent-* 头。
// 这是转发邮件时保留原始头部的标准方式。
func prependResentHeaders(from string, targets []string, original []byte) []byte {
	header := fmt.Sprintf(
		"Resent-From: <%s>\r\nResent-To: %s\r\nResent-Date: %s\r\nX-Forwarded-By: email-bot\r\n",
		from,
		strings.Join(targets, ", "),
		time.Now().Format("Mon, 02 Jan 2006 15:04:05 -0700"),
	)
	out := make([]byte, 0, len(header)+len(original))
	out = append(out, []byte(header)...)
	out = append(out, '\r', '\n') // 添加空行分隔符
	out = append(out, original...)
	return out
}

// sendMailImplicitTLS 通过隐式 TLS 连接（465 端口）发送邮件。
func sendMailImplicitTLS(addr, host string, auth smtp.Auth, from string, to []string, body []byte) error {
	tlsCfg := &tls.Config{ServerName: host, MinVersion: tls.VersionTLS12}

	conn, err := tls.Dial("tcp", addr, tlsCfg)
	if err != nil {
		return fmt.Errorf("TLS 拨号: %w", err)
	}

	c, err := smtp.NewClient(conn, host)
	if err != nil {
		return fmt.Errorf("SMTP 客户端: %w", err)
	}
	defer func() { _ = c.Quit() }()

	if err := c.Auth(auth); err != nil {
		return fmt.Errorf("SMTP 认证: %w", err)
	}
	if err := c.Mail(from); err != nil {
		return fmt.Errorf("SMTP MAIL FROM: %w", err)
	}
	for _, rcpt := range to {
		if err := c.Rcpt(rcpt); err != nil {
			return fmt.Errorf("SMTP RCPT TO %s: %w", rcpt, err)
		}
	}

	wc, err := c.Data()
	if err != nil {
		return fmt.Errorf("SMTP DATA: %w", err)
	}
	if _, err := wc.Write(body); err != nil {
		return fmt.Errorf("SMTP 写入正文: %w", err)
	}
	return wc.Close()
}
